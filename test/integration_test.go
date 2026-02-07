//go:build integration

package test

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compactmapper/internal/converter"
	"compactmapper/internal/sorter"
	"compactmapper/las"
)

// TestFullPipeline tests the complete workflow: sort CSV -> convert to LAS
func TestFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	sortedDir := filepath.Join(tmpDir, "sorted")
	lasDir := filepath.Join(tmpDir, "las")

	// Test CSV with mixed data
	testCSV := filepath.Join(tmpDir, "test.csv")
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:30:02.800,2031878.930,91550.610,5.372,2,10,Design1,?,Data1,Machine1,0.8km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Reverse,Off,?
2025/Oct/01 09:22:49.600,2031879.270,91550.610,5.401,4,10,Design1,?,Data1,Machine1,0.9km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Reverse,Off,?
2025/Oct/01 09:22:49.700,2031879.610,91550.610,5.387,5,10,Design2,?,Data2,Machine2,0.9km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,2.10,?,Reverse,Off,?
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 1: Sort CSV
	if err := sorter.SortCSV(testCSV, sortedDir, false, nil); err != nil {
		t.Fatalf("Sorting failed: %v", err)
	}

	// Verify sorted files exist
	expectedSorted := []string{
		"2025-10-01designDesign1amp097.csv",
		"2025-10-01designDesign2amp210.csv",
	}

	for _, filename := range expectedSorted {
		path := filepath.Join(sortedDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Sorted file %s does not exist", filename)
		}
	}

	// Step 2: Convert sorted CSVs to LAS
	count, err := converter.ConvertDirectory(sortedDir, lasDir)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files converted, got %d", count)
	}

	// Verify LAS files exist and are valid
	expectedLAS := []string{
		"2025-10-01designDesign1amp097.las",
		"2025-10-01designDesign2amp210.las",
	}

	for _, filename := range expectedLAS {
		lasPath := filepath.Join(lasDir, filename)
		if _, err := os.Stat(lasPath); os.IsNotExist(err) {
			t.Errorf("LAS file %s does not exist", filename)
			continue
		}

		// Read and validate LAS file
		reader, err := las.NewReader(lasPath)
		if err != nil {
			t.Errorf("Failed to read LAS file %s: %v", filename, err)
			continue
		}

		header := reader.GetHeader()
		if header.PointFormat != 3 {
			t.Errorf("LAS %s: point format = %d, want 3", filename, header.PointFormat)
		}

		points, err := reader.ReadPoints()
		if err != nil {
			t.Errorf("Failed to read points from %s: %v", filename, err)
			reader.Close()
			continue
		}

		if len(points) == 0 {
			t.Errorf("LAS %s has no points", filename)
		}

		// Verify GPS Time is preserved (non-zero for all points)
		for i, pt := range points {
			if pt.GPSTime == 0 {
				t.Errorf("LAS %s point %d: GPSTime is zero", filename, i)
				break
			}
		}

		reader.Close()
	}
}

// TestMultipleGroupings validates sorting with various Date/DesignName/LastAmp combinations
func TestMultipleGroupings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test.csv")
	sortedDir := filepath.Join(tmpDir, "sorted")

	// CSV with multiple dates, designs, and amplitudes
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,1,10,DesignA,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:01:00.000,1001.0,2001.0,10.1,2,10,DesignA,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:02:00.000,1002.0,2002.0,10.2,1,10,DesignA,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,2.10,?,Forward,Off,?
2025/Oct/01 09:03:00.000,1003.0,2003.0,10.3,1,10,DesignB,?,Data2,M2,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:04:00.000,1004.0,2004.0,10.4,2,10,DesignB,?,Data2,M2,1.0km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,2.10,?,Forward,Off,?
2025/Oct/02 09:00:00.000,1005.0,2005.0,10.5,1,10,DesignA,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/02 09:01:00.000,1006.0,2006.0,10.6,1,10,DesignC,?,Data3,M3,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,1.50,?,Forward,Off,?
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Sort
	if err := sorter.SortCSV(testCSV, sortedDir, false, nil); err != nil {
		t.Fatalf("Sorting failed: %v", err)
	}

	// Expected files: Date x DesignName x LastAmp combinations
	expected := map[string]int{
		"2025-10-01designDesignAamp097.csv": 2, // Oct 1, DesignA, amp 0.97
		"2025-10-01designDesignAamp210.csv": 1, // Oct 1, DesignA, amp 2.10
		"2025-10-01designDesignBamp097.csv": 1, // Oct 1, DesignB, amp 0.97
		"2025-10-01designDesignBamp210.csv": 1, // Oct 1, DesignB, amp 2.10
		"2025-10-02designDesignAamp097.csv": 1, // Oct 2, DesignA, amp 0.97
		"2025-10-02designDesignCamp150.csv": 1, // Oct 2, DesignC, amp 1.50
	}

	// Validate all expected files exist with correct row counts
	for filename, expectedRows := range expected {
		path := filepath.Join(sortedDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", filename)
			continue
		}

		actualRows, err := countCSVRows(path)
		if err != nil {
			t.Errorf("Error reading file %s: %v", filename, err)
			continue
		}

		if actualRows != expectedRows {
			t.Errorf("File %s: got %d rows, want %d", filename, actualRows, expectedRows)
		}
	}

	// Verify no unexpected files
	actualFiles, err := filepath.Glob(filepath.Join(sortedDir, "*.csv"))
	if err != nil {
		t.Fatal(err)
	}

	if len(actualFiles) != len(expected) {
		t.Errorf("Created %d files, expected %d", len(actualFiles), len(expected))
	}
}

// TestLargeDataChunking validates sorting works correctly with chunking (10K+ rows)
func TestLargeDataChunking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test.csv")
	sortedDir := filepath.Join(tmpDir, "sorted")

	// Generate CSV with 15,000 rows to test chunking (ChunkSize = 10,000)
	var csvBuilder strings.Builder
	csvBuilder.WriteString("Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp\n")

	const totalRows = 15000
	for i := 0; i < totalRows; i++ {
		design := "DesignA"
		if i%2 == 0 {
			design = "DesignB"
		}
		// Calculate time: increment by 1 second per row
		hour := 9 + (i / 3600)
		minute := (i % 3600) / 60
		second := i % 60
		csvBuilder.WriteString(fmt.Sprintf(
			"2025/Oct/01 %02d:%02d:%02d.000,%f,%f,%f,%d,10,%s,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?\n",
			hour, minute, second, 1000.0+float64(i), 2000.0+float64(i), 10.0+float64(i)*0.01, i%5+1, design,
		))
	}

	if err := os.WriteFile(testCSV, []byte(csvBuilder.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// Sort
	if err := sorter.SortCSV(testCSV, sortedDir, false, nil); err != nil {
		t.Fatalf("Sorting failed: %v", err)
	}

	// Verify files were created
	files, err := filepath.Glob(filepath.Join(sortedDir, "*.csv"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("No sorted files were created")
	}

	// Validate total row count matches input
	totalOutputRows := 0
	for _, file := range files {
		rows, err := countCSVRows(file)
		if err != nil {
			t.Errorf("Error reading file %s: %v", filepath.Base(file), err)
			continue
		}
		totalOutputRows += rows
	}

	if totalOutputRows != totalRows {
		t.Errorf("Total output rows: got %d, want %d", totalOutputRows, totalRows)
	}

	t.Logf("Successfully processed %d rows across %d files", totalOutputRows, len(files))
}

// TestEdgeCases validates handling of special characters, BOM, and missing values
func TestEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		csvContent string
		wantFiles  int
		wantError  bool
	}{
		{
			name: "UTF-8 BOM",
			csvContent: "\xEF\xBB\xBF" + `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,1,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
`,
			wantFiles: 1,
			wantError: false,
		},
		{
			name: "Special characters in DesignName",
			csvContent: `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,1,10,Design_A-123,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:01:00.000,1001.0,2001.0,10.1,2,10,Design/B*456,?,Data2,M2,1.0km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,1.50,?,Forward,Off,?
`,
			wantFiles: 2,
			wantError: false,
		},
		{
			name: "Missing LastAmp values",
			csvContent: `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,1,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,?,?,Forward,Off,?
2025/Oct/01 09:01:00.000,1001.0,2001.0,10.1,2,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,?,?,Forward,Off,?
`,
			wantFiles: 1,
			wantError: false,
		},
		{
			name: "Question marks as missing values",
			csvContent: `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,1,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,?,?,Forward,Off,?
`,
			wantFiles: 1,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testCSV := filepath.Join(tmpDir, "test.csv")
			sortedDir := filepath.Join(tmpDir, "sorted")

			if err := os.WriteFile(testCSV, []byte(tt.csvContent), 0644); err != nil {
				t.Fatal(err)
			}

			err := sorter.SortCSV(testCSV, sortedDir, false, nil)
			if tt.wantError && err == nil {
				t.Error("Expected error, got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantError {
				files, err := filepath.Glob(filepath.Join(sortedDir, "*.csv"))
				if err != nil {
					t.Fatal(err)
				}

				if len(files) != tt.wantFiles {
					t.Errorf("Created %d files, want %d", len(files), tt.wantFiles)
				}
			}
		})
	}
}

// TestLASFormatValidation validates LAS 1.2 format compliance
func TestLASFormatValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "test.csv")
	sortedDir := filepath.Join(tmpDir, "sorted")
	lasDir := filepath.Join(tmpDir, "las")

	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:00:00.000,1000.0,2000.0,10.0,2,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:01:00.000,1001.0,2001.0,10.1,4,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,4,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
2025/Oct/01 09:02:00.000,1002.0,2002.0,10.2,6,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,6,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Sort and convert
	if err := sorter.SortCSV(testCSV, sortedDir, false, nil); err != nil {
		t.Fatalf("Sorting failed: %v", err)
	}

	if _, err := converter.ConvertDirectory(sortedDir, lasDir); err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Validate LAS format
	lasFiles, err := filepath.Glob(filepath.Join(lasDir, "*.las"))
	if err != nil {
		t.Fatal(err)
	}

	if len(lasFiles) == 0 {
		t.Fatal("No LAS files created")
	}

	for _, lasPath := range lasFiles {
		reader, err := las.NewReader(lasPath)
		if err != nil {
			t.Errorf("Failed to read LAS file %s: %v", filepath.Base(lasPath), err)
			continue
		}

		header := reader.GetHeader()

		// Validate LAS version
		if header.VersionMajor != 1 || header.VersionMinor != 2 {
			t.Errorf("LAS version: got %d.%d, want 1.2", header.VersionMajor, header.VersionMinor)
		}

		// Validate point format (Format 3 = XYZ + GPS Time + RGB)
		if header.PointFormat != 3 {
			t.Errorf("Point format: got %d, want 3 (XYZ + GPS Time + RGB)", header.PointFormat)
		}

		// Validate point record length (format 3 = 34 bytes)
		if header.PointRecordLength != 34 {
			t.Errorf("Point record length: got %d, want 34", header.PointRecordLength)
		}

		// Read and validate points
		points, err := reader.ReadPoints()
		if err != nil {
			t.Errorf("Failed to read points: %v", err)
		}

		if len(points) == 0 {
			t.Error("LAS file has no points")
		}

		// Validate RGB values exist and are in valid range
		for i, point := range points {
			if point.R == 0 && point.G == 0 && point.B == 0 {
				t.Errorf("Point %d has no RGB values", i)
			}
			if point.R > 65535 || point.G > 65535 || point.B > 65535 {
				t.Errorf("Point %d RGB values out of range (0-65535)", i)
			}
		}

		// Validate GPS Time is present (Format 3 feature)
		for i, point := range points {
			if point.GPSTime == 0 {
				t.Errorf("Point %d has no GPS Time (Format 3 requires GPS Time)", i)
				break
			}
		}

		reader.Close()
	}
}

// TestErrorHandling validates error handling for invalid CSV input
func TestErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		csvContent string
		wantError  bool
	}{
		{
			name:       "Missing Time column",
			csvContent: `CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp`,
			wantError:  true,
		},
		{
			name:       "Missing DesignName column",
			csvContent: `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp`,
			wantError:  true,
		},
		{
			name: "Invalid date format",
			csvContent: `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025-10-01 09:00:00,1000.0,2000.0,10.0,1,10,Design1,?,Data1,M1,1.0km/h,RTK Fixed,Coarse (0.100),4,1,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,Off,?`,
			wantError: true,
		},
		{
			name:       "Empty file",
			csvContent: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testCSV := filepath.Join(tmpDir, "test.csv")
			sortedDir := filepath.Join(tmpDir, "sorted")

			if err := os.WriteFile(testCSV, []byte(tt.csvContent), 0644); err != nil {
				t.Fatal(err)
			}

			err := sorter.SortCSV(testCSV, sortedDir, false, nil)

			if tt.wantError && err == nil {
				t.Error("Expected error, got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function to count CSV rows
func countCSVRows(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}

	// Subtract 1 for header row
	return len(records) - 1, nil
}
