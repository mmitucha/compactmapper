package las

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// Header represents LAS file header information
type Header struct {
	VersionMajor      uint8
	VersionMinor      uint8
	PointFormat       uint8
	PointCount        uint32
	PointRecordLength uint16
	XScale, YScale, ZScale float64
	XOffset, YOffset, ZOffset float64
	MinX, MinY, MinZ float64
	MaxX, MaxY, MaxZ float64
}

// Reader handles reading LAS files
type Reader struct {
	file   *os.File
	header Header
}

// NewReader creates a new LAS reader
func NewReader(filename string) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	reader := &Reader{file: file}

	// Read and parse header
	if err := reader.readHeader(); err != nil {
		file.Close()
		return nil, err
	}

	return reader, nil
}

// Close closes the reader
func (r *Reader) Close() error {
	return r.file.Close()
}

// readHeader reads and parses the LAS header
func (r *Reader) readHeader() error {
	header := make([]byte, 227)
	if _, err := io.ReadFull(r.file, header); err != nil {
		return fmt.Errorf("error reading header: %v", err)
	}

	// Verify signature
	if string(header[0:4]) != "LASF" {
		return fmt.Errorf("invalid LAS file: wrong signature")
	}

	// Parse version
	r.header.VersionMajor = header[24]
	r.header.VersionMinor = header[25]

	// Parse point format and count
	r.header.PointFormat = header[104]
	r.header.PointRecordLength = binary.LittleEndian.Uint16(header[105:107])
	r.header.PointCount = binary.LittleEndian.Uint32(header[107:111])

	// Parse scale factors
	r.header.XScale = math.Float64frombits(binary.LittleEndian.Uint64(header[131:139]))
	r.header.YScale = math.Float64frombits(binary.LittleEndian.Uint64(header[139:147]))
	r.header.ZScale = math.Float64frombits(binary.LittleEndian.Uint64(header[147:155]))

	// Parse offsets
	r.header.XOffset = math.Float64frombits(binary.LittleEndian.Uint64(header[155:163]))
	r.header.YOffset = math.Float64frombits(binary.LittleEndian.Uint64(header[163:171]))
	r.header.ZOffset = math.Float64frombits(binary.LittleEndian.Uint64(header[171:179]))

	// Parse bounds
	r.header.MaxX = math.Float64frombits(binary.LittleEndian.Uint64(header[179:187]))
	r.header.MaxY = math.Float64frombits(binary.LittleEndian.Uint64(header[187:195]))
	r.header.MaxZ = math.Float64frombits(binary.LittleEndian.Uint64(header[195:203]))

	r.header.MinX = math.Float64frombits(binary.LittleEndian.Uint64(header[203:211]))
	r.header.MinY = math.Float64frombits(binary.LittleEndian.Uint64(header[211:219]))
	r.header.MinZ = math.Float64frombits(binary.LittleEndian.Uint64(header[219:227]))

	return nil
}

// GetHeader returns the header information
func (r *Reader) GetHeader() Header {
	return r.header
}

// ReadPoints reads all points from the LAS file
func (r *Reader) ReadPoints() ([]Point, error) {
	// Seek to start of point data (after header)
	if _, err := r.file.Seek(227, 0); err != nil {
		return nil, fmt.Errorf("error seeking to point data: %v", err)
	}

	switch r.header.PointFormat {
	case 2:
		return r.readPointsFormat2()
	case 3:
		return r.readPointsFormat3()
	default:
		return nil, fmt.Errorf("unsupported point format: %d (supported: 2, 3)", r.header.PointFormat)
	}
}

// readPointsFormat2 reads Format 2 points (26 bytes: XYZ + Intensity + RGB)
func (r *Reader) readPointsFormat2() ([]Point, error) {
	points := make([]Point, 0, r.header.PointCount)

	for i := uint32(0); i < r.header.PointCount; i++ {
		pointData := make([]byte, 26)
		if _, err := io.ReadFull(r.file, pointData); err != nil {
			return nil, fmt.Errorf("error reading point %d: %v", i, err)
		}

		x := int32(binary.LittleEndian.Uint32(pointData[0:4]))
		y := int32(binary.LittleEndian.Uint32(pointData[4:8]))
		z := int32(binary.LittleEndian.Uint32(pointData[8:12]))

		points = append(points, Point{
			X:              float64(x)*r.header.XScale + r.header.XOffset,
			Y:              float64(y)*r.header.YScale + r.header.YOffset,
			Z:              float64(z)*r.header.ZScale + r.header.ZOffset,
			Intensity:      binary.LittleEndian.Uint16(pointData[12:14]),
			Classification: pointData[15],
			R:              binary.LittleEndian.Uint16(pointData[20:22]),
			G:              binary.LittleEndian.Uint16(pointData[22:24]),
			B:              binary.LittleEndian.Uint16(pointData[24:26]),
		})
	}

	return points, nil
}

// readPointsFormat3 reads Format 3 points (34 bytes: XYZ + Intensity + GPS Time + RGB)
func (r *Reader) readPointsFormat3() ([]Point, error) {
	points := make([]Point, 0, r.header.PointCount)

	for i := uint32(0); i < r.header.PointCount; i++ {
		pointData := make([]byte, 34)
		if _, err := io.ReadFull(r.file, pointData); err != nil {
			return nil, fmt.Errorf("error reading point %d: %v", i, err)
		}

		x := int32(binary.LittleEndian.Uint32(pointData[0:4]))
		y := int32(binary.LittleEndian.Uint32(pointData[4:8]))
		z := int32(binary.LittleEndian.Uint32(pointData[8:12]))

		points = append(points, Point{
			X:              float64(x)*r.header.XScale + r.header.XOffset,
			Y:              float64(y)*r.header.YScale + r.header.YOffset,
			Z:              float64(z)*r.header.ZScale + r.header.ZOffset,
			Intensity:      binary.LittleEndian.Uint16(pointData[12:14]),
			Classification: pointData[15],
			GPSTime:        math.Float64frombits(binary.LittleEndian.Uint64(pointData[20:28])),
			R:              binary.LittleEndian.Uint16(pointData[28:30]),
			G:              binary.LittleEndian.Uint16(pointData[30:32]),
			B:              binary.LittleEndian.Uint16(pointData[32:34]),
		})
	}

	return points, nil
}
