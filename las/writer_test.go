package las

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteAndRead tests writing and reading a LAS file with Format 3 (GPS Time + RGB)
func TestWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	lasFile := filepath.Join(tmpDir, "test.las")

	// Create test points with GPS Time values
	testPoints := []Point{
		{X: 100.5, Y: 200.3, Z: 5.2, R: 65535, G: 0, B: 0, Intensity: 100, Classification: 1, GPSTime: 1727776202.800},
		{X: 100.6, Y: 200.4, Z: 5.3, R: 0, G: 65535, B: 0, Intensity: 150, Classification: 2, GPSTime: 1727776203.000},
		{X: 100.7, Y: 200.5, Z: 5.4, R: 0, G: 0, B: 65535, Intensity: 200, Classification: 3, GPSTime: 1727776203.200},
	}

	// Write LAS file
	writer := NewWriter()
	for _, pt := range testPoints {
		writer.AddPoint(pt)
	}

	if err := writer.Write(lasFile); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file size: 227 header + 3 points * 34 bytes (Format 3) = 329
	info, err := os.Stat(lasFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	expectedSize := int64(227 + len(testPoints)*34)
	if info.Size() != expectedSize {
		t.Errorf("File size = %d, want %d (227 header + %d * 34)", info.Size(), expectedSize, len(testPoints))
	}

	// Read LAS file back
	reader, err := NewReader(lasFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	// Check header
	header := reader.GetHeader()
	if header.PointFormat != 3 {
		t.Errorf("Point format = %d, want 3", header.PointFormat)
	}
	if header.PointRecordLength != 34 {
		t.Errorf("Point record length = %d, want 34", header.PointRecordLength)
	}
	if header.PointCount != uint32(len(testPoints)) {
		t.Errorf("Point count = %d, want %d", header.PointCount, len(testPoints))
	}
	if header.VersionMajor != 1 || header.VersionMinor != 2 {
		t.Errorf("Version = %d.%d, want 1.2", header.VersionMajor, header.VersionMinor)
	}

	// Read points
	points, err := reader.ReadPoints()
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	if len(points) != len(testPoints) {
		t.Fatalf("Read %d points, expected %d", len(points), len(testPoints))
	}

	// Compare points (with tolerance for floating point precision loss from scaling)
	tolerance := 0.01
	for i, got := range points {
		want := testPoints[i]

		if math.Abs(got.X-want.X) > tolerance {
			t.Errorf("Point %d: X = %f, want %f", i, got.X, want.X)
		}
		if math.Abs(got.Y-want.Y) > tolerance {
			t.Errorf("Point %d: Y = %f, want %f", i, got.Y, want.Y)
		}
		if math.Abs(got.Z-want.Z) > tolerance {
			t.Errorf("Point %d: Z = %f, want %f", i, got.Z, want.Z)
		}
		if got.R != want.R || got.G != want.G || got.B != want.B {
			t.Errorf("Point %d: RGB = (%d,%d,%d), want (%d,%d,%d)",
				i, got.R, got.G, got.B, want.R, want.G, want.B)
		}
		if got.Intensity != want.Intensity {
			t.Errorf("Point %d: Intensity = %d, want %d", i, got.Intensity, want.Intensity)
		}
		if got.Classification != want.Classification {
			t.Errorf("Point %d: Classification = %d, want %d", i, got.Classification, want.Classification)
		}
		if math.Abs(got.GPSTime-want.GPSTime) > 0.001 {
			t.Errorf("Point %d: GPSTime = %f, want %f", i, got.GPSTime, want.GPSTime)
		}
	}
}

// TestWriteEmptyFile tests error handling for empty point list
func TestWriteEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	lasFile := filepath.Join(tmpDir, "empty.las")

	writer := NewWriter()
	err := writer.Write(lasFile)
	if err == nil {
		t.Error("Expected error for empty point list, got nil")
	}
}

// TestBoundsCalculation tests min/max bounds calculation
func TestBoundsCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	lasFile := filepath.Join(tmpDir, "bounds.las")

	// Create points with known bounds
	points := []Point{
		{X: 10.0, Y: 20.0, Z: 1.0, R: 0, G: 0, B: 0, Intensity: 0, Classification: 1},
		{X: 50.0, Y: 60.0, Z: 5.0, R: 0, G: 0, B: 0, Intensity: 0, Classification: 1},
		{X: 30.0, Y: 40.0, Z: 3.0, R: 0, G: 0, B: 0, Intensity: 0, Classification: 1},
	}

	writer := NewWriter()
	for _, pt := range points {
		writer.AddPoint(pt)
	}

	if err := writer.Write(lasFile); err != nil {
		t.Fatal(err)
	}

	// Read back and check bounds
	reader, err := NewReader(lasFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()

	header := reader.GetHeader()

	// Expected bounds
	if header.MinX != 10.0 || header.MaxX != 50.0 {
		t.Errorf("X bounds = [%f, %f], want [10, 50]", header.MinX, header.MaxX)
	}
	if header.MinY != 20.0 || header.MaxY != 60.0 {
		t.Errorf("Y bounds = [%f, %f], want [20, 60]", header.MinY, header.MaxY)
	}
	if header.MinZ != 1.0 || header.MaxZ != 5.0 {
		t.Errorf("Z bounds = [%f, %f], want [1, 5]", header.MinZ, header.MaxZ)
	}
}

// TestReadInvalidFile tests error handling for invalid files
func TestReadInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.las")

	// Create invalid LAS file
	if err := os.WriteFile(invalidFile, []byte("NOT A LAS FILE"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewReader(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid LAS file, got nil")
	}
}

// TestReadNonexistentFile tests error handling for missing files
func TestReadNonexistentFile(t *testing.T) {
	_, err := NewReader("/nonexistent/file.las")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}
