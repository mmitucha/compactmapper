package converter

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"compactmapper/las"
)

// TestConvertCSVToLAS tests basic CSV to LAS conversion verifying
// that CSV fields are correctly mapped to LAS point data
func TestConvertCSVToLAS(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Create test CSV with known values:
	// Row 1: PassCount(1) < TargPassCount(4) → Red
	// Row 2: PassCount(2) < TargPassCount(4) → Red
	// Row 3: PassCount(4) == TargPassCount(4) → Green
	// Row 4: PassCount(5) > TargPassCount(4) → Blue
	testCSV := filepath.Join(tmpDir, "test.csv")
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,TargPassCount
2025/Oct/01 09:30:02.800,100.5,200.3,5.2,1,4
2025/Oct/01 09:30:03.000,100.6,200.4,5.3,2,4
2025/Oct/01 09:30:03.200,100.7,200.5,5.4,4,4
2025/Oct/01 09:30:03.400,100.8,200.6,5.5,5,4
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Convert
	err := ConvertCSVToLAS(testCSV, outputDir)
	if err != nil {
		t.Fatalf("ConvertCSVToLAS failed: %v", err)
	}

	// Read back LAS file and verify content
	lasFile := filepath.Join(outputDir, "test.las")
	reader, err := las.NewReader(lasFile)
	if err != nil {
		t.Fatalf("Failed to read LAS file: %v", err)
	}
	defer func() { _ = reader.Close() }()

	header := reader.GetHeader()
	if header.PointFormat != 3 {
		t.Errorf("Point format = %d, want 3", header.PointFormat)
	}
	if header.PointCount != 4 {
		t.Errorf("Point count = %d, want 4", header.PointCount)
	}

	points, err := reader.ReadPoints()
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	if len(points) != 4 {
		t.Fatalf("Got %d points, want 4", len(points))
	}

	// Verify coordinates (CellE_m → X, CellN_m → Y, Elevation_m → Z)
	tolerance := 0.01
	expectedCoords := [][3]float64{
		{200.3, 100.5, 5.2},
		{200.4, 100.6, 5.3},
		{200.5, 100.7, 5.4},
		{200.6, 100.8, 5.5},
	}
	for i, want := range expectedCoords {
		got := points[i]
		if math.Abs(got.X-want[0]) > tolerance || math.Abs(got.Y-want[1]) > tolerance || math.Abs(got.Z-want[2]) > tolerance {
			t.Errorf("Point %d: coords = (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)",
				i, got.X, got.Y, got.Z, want[0], want[1], want[2])
		}
	}

	// Verify color mapping: Red(under), Red(under), Green(at), Blue(over)
	expectedColors := [][3]uint16{
		{65535, 0, 0},     // PassCount 1 < 4 → Red
		{65535, 0, 0},     // PassCount 2 < 4 → Red
		{0, 65535, 0},     // PassCount 4 == 4 → Green
		{0, 0, 65535},     // PassCount 5 > 4 → Blue
	}
	for i, want := range expectedColors {
		got := points[i]
		if got.R != want[0] || got.G != want[1] || got.B != want[2] {
			t.Errorf("Point %d: RGB = (%d,%d,%d), want (%d,%d,%d)",
				i, got.R, got.G, got.B, want[0], want[1], want[2])
		}
	}

	// Verify GPS Time is set (non-zero, derived from CSV timestamps)
	for i, pt := range points {
		if pt.GPSTime == 0 {
			t.Errorf("Point %d: GPSTime is zero, expected timestamp from CSV", i)
		}
	}
	// GPS times should be monotonically increasing (timestamps are sequential)
	for i := 1; i < len(points); i++ {
		if points[i].GPSTime <= points[i-1].GPSTime {
			t.Errorf("Point %d: GPSTime %.3f should be > Point %d GPSTime %.3f",
				i, points[i].GPSTime, i-1, points[i-1].GPSTime)
		}
	}
}

// TestConvertDirectory tests converting all CSV files in a directory
func TestConvertDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	// Create input directory
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create multiple test CSVs
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,TargPassCount
2025/Oct/01 09:30:02.800,100,200,5.0,2,4
`
	for i := 1; i <= 3; i++ {
		testCSV := filepath.Join(inputDir, filepath.Base(filepath.Join("", "test"+string(rune('0'+i))+".csv")))
		if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Convert all
	count, err := ConvertDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ConvertDirectory failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files converted, got %d", count)
	}

	// Check all LAS files exist
	for i := 1; i <= 3; i++ {
		lasFile := filepath.Join(outputDir, filepath.Base(filepath.Join("", "test"+string(rune('0'+i))+".las")))
		if _, err := os.Stat(lasFile); os.IsNotExist(err) {
			t.Errorf("LAS file %s not created", lasFile)
		}
	}
}

// TestColorAssignment tests PassCount-based color assignment
func TestColorAssignment(t *testing.T) {
	tests := []struct {
		passCount  int
		targPass   int
		wantR      uint16
		wantG      uint16
		wantB      uint16
		colorName  string
	}{
		{1, 4, 65535, 0, 0, "Red (under target)"},
		{2, 4, 65535, 0, 0, "Red (under target)"},
		{4, 4, 0, 65535, 0, "Green (at target)"},
		{5, 4, 0, 0, 65535, "Blue (over target)"},
		{10, 4, 0, 0, 65535, "Blue (over target)"},
	}

	for _, tt := range tests {
		t.Run(tt.colorName, func(t *testing.T) {
			r, g, b := determineColor(tt.passCount, tt.targPass)
			if r != tt.wantR || g != tt.wantG || b != tt.wantB {
				t.Errorf("determineColor(%d, %d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.passCount, tt.targPass, r, g, b, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

// TestConvertMissingColumns tests error handling for missing columns
func TestConvertMissingColumns(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "bad.csv")
	outputDir := filepath.Join(tmpDir, "output")

	// Create CSV without required columns
	csvContent := "Time,X,Y\n2025/Oct/01 09:30:02.800,100,200\n"
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return error
	err := ConvertCSVToLAS(testCSV, outputDir)
	if err == nil {
		t.Error("Expected error for missing columns, got nil")
	}
}

// TestConvertInvalidData tests error handling for invalid numeric data
func TestConvertInvalidData(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "invalid.csv")
	outputDir := filepath.Join(tmpDir, "output")

	// Create CSV with invalid numeric values
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,TargPassCount
2025/Oct/01 09:30:02.800,abc,200,5.0,2,4
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return error
	err := ConvertCSVToLAS(testCSV, outputDir)
	if err == nil {
		t.Error("Expected error for invalid numeric data, got nil")
	}
}

// TestConvertEmptyCSV tests handling of empty CSV files
func TestConvertEmptyCSV(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "empty.csv")
	outputDir := filepath.Join(tmpDir, "output")

	// Create CSV with only header
	csvContent := "Time,CellN_m,CellE_m,Elevation_m,PassCount,TargPassCount\n"
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return error (no points to write)
	err := ConvertCSVToLAS(testCSV, outputDir)
	if err == nil {
		t.Error("Expected error for empty CSV, got nil")
	}
}
