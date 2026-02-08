package las

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"
)

// Point represents a LAS point with RGB color and GPS Time
type Point struct {
	X, Y, Z    float64
	Intensity  uint16
	R, G, B    uint16
	Classification uint8
	GPSTime    float64 // GPS time in seconds (Adjusted Standard GPS Time)
}

// Writer handles LAS file creation
type Writer struct {
	points []Point
	minX, minY, minZ float64
	maxX, maxY, maxZ float64
}

// NewWriter creates a new LAS writer
func NewWriter() *Writer {
	return &Writer{
		points: make([]Point, 0),
		minX: math.MaxFloat64, minY: math.MaxFloat64, minZ: math.MaxFloat64,
		maxX: -math.MaxFloat64, maxY: -math.MaxFloat64, maxZ: -math.MaxFloat64,
	}
}

// AddPoint adds a point to the writer
func (w *Writer) AddPoint(p Point) {
	w.points = append(w.points, p)

	// Update bounds
	if p.X < w.minX { w.minX = p.X }
	if p.Y < w.minY { w.minY = p.Y }
	if p.Z < w.minZ { w.minZ = p.Z }
	if p.X > w.maxX { w.maxX = p.X }
	if p.Y > w.maxY { w.maxY = p.Y }
	if p.Z > w.maxZ { w.maxZ = p.Z }
}

// Write writes the LAS file to disk
func (w *Writer) Write(filename string) (retErr error) {
	if len(w.points) == 0 {
		return fmt.Errorf("no points to write")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	// Propagate close error via named return: Close() flushes OS buffers,
	// so a failure here means the LAS file is silently corrupt.
	// We only overwrite retErr when no prior write error exists.
	defer func() {
		if cerr := file.Close(); retErr == nil && cerr != nil {
			retErr = fmt.Errorf("error closing LAS file: %w", cerr)
		}
	}()

	// LAS 1.2 Header (227 bytes)
	header := make([]byte, 227)

	// File signature "LASF"
	copy(header[0:4], []byte("LASF"))

	// File source ID
	binary.LittleEndian.PutUint16(header[4:6], 0)

	// Global encoding (GPS Time Type: 1 = Adjusted Standard GPS Time)
	binary.LittleEndian.PutUint16(header[6:8], 1)

	// Project ID (GUID) - zeros
	// header[8:24] already zeros

	// Version Major = 1, Minor = 2
	header[24] = 1
	header[25] = 2

	// System Identifier (32 bytes)
	copy(header[26:58], []byte("CompactMapper"))

	// Generating Software (32 bytes)
	copy(header[58:90], []byte("CompactMapper v1.0"))

	// File Creation Day of Year & Year
	now := time.Now()
	dayOfYear := now.YearDay()
	binary.LittleEndian.PutUint16(header[90:92], uint16(dayOfYear))
	binary.LittleEndian.PutUint16(header[92:94], uint16(now.Year()))

	// Header size
	binary.LittleEndian.PutUint16(header[94:96], 227)

	// Offset to point data
	binary.LittleEndian.PutUint32(header[96:100], 227)

	// Number of Variable Length Records
	binary.LittleEndian.PutUint32(header[100:104], 0)

	// Point Data Format ID (3 = XYZ + Intensity + GPS Time + RGB)
	header[104] = 3

	// Point Data Record Length (34 bytes for format 3)
	binary.LittleEndian.PutUint16(header[105:107], 34)

	// Number of point records
	binary.LittleEndian.PutUint32(header[107:111], uint32(len(w.points)))

	// Number of points by return (5 fields)
	// We'll put all points in return 1
	binary.LittleEndian.PutUint32(header[111:115], uint32(len(w.points)))

	// Scale factors (0.001 for better precision)
	xScale := 0.001
	yScale := 0.001
	zScale := 0.001
	binary.LittleEndian.PutUint64(header[131:139], math.Float64bits(xScale))
	binary.LittleEndian.PutUint64(header[139:147], math.Float64bits(yScale))
	binary.LittleEndian.PutUint64(header[147:155], math.Float64bits(zScale))

	// Offsets
	binary.LittleEndian.PutUint64(header[155:163], math.Float64bits(w.minX))
	binary.LittleEndian.PutUint64(header[163:171], math.Float64bits(w.minY))
	binary.LittleEndian.PutUint64(header[171:179], math.Float64bits(w.minZ))

	// Max X, Y, Z
	binary.LittleEndian.PutUint64(header[179:187], math.Float64bits(w.maxX))
	binary.LittleEndian.PutUint64(header[187:195], math.Float64bits(w.maxY))
	binary.LittleEndian.PutUint64(header[195:203], math.Float64bits(w.maxZ))

	// Min X, Y, Z
	binary.LittleEndian.PutUint64(header[203:211], math.Float64bits(w.minX))
	binary.LittleEndian.PutUint64(header[211:219], math.Float64bits(w.minY))
	binary.LittleEndian.PutUint64(header[219:227], math.Float64bits(w.minZ))

	// Write header
	if _, err := file.Write(header); err != nil {
		return err
	}

	// Write point data (Format 3: 34 bytes per point)
	for _, p := range w.points {
		pointData := make([]byte, 34)

		// X, Y, Z as scaled integers
		x := int32((p.X - w.minX) / xScale)
		y := int32((p.Y - w.minY) / yScale)
		z := int32((p.Z - w.minZ) / zScale)

		binary.LittleEndian.PutUint32(pointData[0:4], uint32(x))
		binary.LittleEndian.PutUint32(pointData[4:8], uint32(y))
		binary.LittleEndian.PutUint32(pointData[8:12], uint32(z))

		// Intensity
		binary.LittleEndian.PutUint16(pointData[12:14], p.Intensity)

		// Return number, number of returns, scan direction, edge of flight line
		pointData[14] = 0x01 // First return of 1

		// Classification
		pointData[15] = p.Classification

		// Scan angle rank
		pointData[16] = 0

		// User data
		pointData[17] = 0

		// Point source ID
		binary.LittleEndian.PutUint16(pointData[18:20], 0)

		// GPS Time (Format 3) - 8 bytes at offset 20
		binary.LittleEndian.PutUint64(pointData[20:28], math.Float64bits(p.GPSTime))

		// RGB (Format 3) - moved to offset 28
		binary.LittleEndian.PutUint16(pointData[28:30], p.R)
		binary.LittleEndian.PutUint16(pointData[30:32], p.G)
		binary.LittleEndian.PutUint16(pointData[32:34], p.B)

		if _, err := file.Write(pointData); err != nil {
			return err
		}
	}

	return nil
}
