package sorter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSanitizeFilename tests filename sanitization
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple.csv", "simple.csv"},
		{"file:with:colons.csv", "filewithcolons.csv"},
		{"file<with>brackets.csv", "filewithbrackets.csv"},
		{"file/with\\slashes.csv", "filewithslashes.csv"},
		{"file|with*special?.csv", "filewithspecial.csv"},
		{`file"with"quotes.csv`, "filewithquotes.csv"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseDate tests date extraction from Time field
func TestParseDate(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		shouldError bool
	}{
		{"2025/Oct/01 09:30:02.800", "2025-10-01", false},
		{"2025/Oct/01 09:22:49.600", "2025-10-01", false},
		{"2024/Dec/31 23:59:59.999", "2024-12-31", false},
		{"2024/Jan/01 00:00:00.000", "2024-01-01", false},
		{"invalid", "", true},
		{"2025-10-01", "", true}, // Wrong format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.shouldError {
				if err == nil {
					t.Errorf("parseDate(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseDate(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseDate(%q) = %q, want %q", tt.input, result, tt.expected)
				}
			}
		})
	}
}

// TestNormalizeAmp tests amplitude normalization
func TestNormalizeAmp(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.97", "097"},
		{"0.55", "055"},
		{"2.10", "210"},
		{"1.5", "15"},
		{"", "no_amp"},
		{"?", "no_amp"},
		{"0.9789", "097"}, // Should truncate to 3 chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeAmp(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeAmp(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGenerateFilename tests filename generation
func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		date     string
		design   string
		amp      string
		expected string
	}{
		{
			"2025-10-01",
			"1010_NY_avrett_LAV_Planum-065",
			"097",
			"2025-10-01design1010_NY_avrett_LAV_Planum-065amp097.csv",
		},
		{
			"2025-10-01",
			"1010_NY_avrett_LAV_Planum-065",
			"no_amp",
			"2025-10-01design1010_NY_avrett_LAV_Planum-065amp.csv",
		},
		{
			"2024-12-31",
			"Simple_Design",
			"210",
			"2024-12-31designSimple_Designamp210.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := generateFilename(tt.date, tt.design, tt.amp)
			if result != tt.expected {
				t.Errorf("generateFilename(%q, %q, %q) = %q, want %q",
					tt.date, tt.design, tt.amp, result, tt.expected)
			}
		})
	}
}

// TestSortCSV tests the main sorting functionality
func TestSortCSV(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Create test CSV with mixed data
	testCSV := filepath.Join(tmpDir, "test.csv")
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,LastRadioLtncy,DesignName,Task,MeasuredData,Machine,Speed,LastGPSMode,GPSAccTol,TargPassCount,TotalPasses,Lift,LastCMV,TargCMV,LastEVIB1,TargEVIB1,LastEVIB2,TargEVIB2,LastMDP,TargMDP,LastRMV,LastFreq,LastAmp,TargThickness,MachineGear,VibeState,LastTemp
2025/Oct/01 09:30:02.800,2031878.930,91550.610,5.372,2,10,Design1,?,Data1,Machine1,0.8km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Reverse,Off,?
2025/Oct/01 09:22:49.600,2031879.270,91550.610,5.401,2,10,Design1,?,Data1,Machine1,0.9km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Reverse,Off,?
2025/Oct/01 09:22:49.700,2031879.610,91550.610,5.387,2,10,Design2,?,Data2,Machine2,0.9km/h,RTK Fixed,Coarse (0.100),4,2,?,?,50.0,?,?,?,?,?,50.0,?,?,2.10,?,Reverse,Off,?
2025/Oct/02 10:00:00.000,2031880.000,91551.000,5.400,3,10,Design1,?,Data3,Machine1,1.0km/h,RTK Fixed,Coarse (0.100),4,3,?,?,50.0,?,?,?,?,?,50.0,?,?,0.97,?,Forward,On,?
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run sorter
	err := SortCSV(testCSV, outputDir, false, nil)
	if err != nil {
		t.Fatalf("SortCSV failed: %v", err)
	}

	// Check output files exist
	expectedFiles := []string{
		"2025-10-01designDesign1amp097.csv",
		"2025-10-01designDesign2amp210.csv",
		"2025-10-02designDesign1amp097.csv",
	}

	for _, filename := range expectedFiles {
		path := filepath.Join(outputDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", filename)
		}
	}

	// Verify content of one file
	content, err := os.ReadFile(filepath.Join(outputDir, "2025-10-01designDesign1amp097.csv"))
	if err != nil {
		t.Fatal(err)
	}

	// Should have header + 2 data rows
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("Expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Verify header is present
	if !strings.HasPrefix(lines[0], "Time,") {
		t.Errorf("Expected header to start with 'Time,', got: %s", lines[0])
	}
}

// TestSortCSVDirectory tests sorting from a directory
func TestSortCSVDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	// Create input directory
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test CSV
	testCSV := filepath.Join(inputDir, "test.csv")
	csvContent := `Time,CellN_m,CellE_m,Elevation_m,PassCount,DesignName,LastAmp,TargPassCount
2025/Oct/01 09:30:02.800,100,200,5.0,2,TestDesign,0.55,4
`
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Run sorter on directory
	err := SortCSVDirectory(inputDir, outputDir, false, nil)
	if err != nil {
		t.Fatalf("SortCSVDirectory failed: %v", err)
	}

	// Check output file exists
	expectedFile := filepath.Join(outputDir, "2025-10-01designTestDesignamp055.csv")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Error("Expected output file does not exist")
	}
}

// TestSortCSVWithChunking tests handling of large files with chunking
func TestSortCSVWithChunking(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	// Create CSV with more rows than chunk size
	testCSV := filepath.Join(tmpDir, "large.csv")
	f, err := os.Create(testCSV)
	if err != nil {
		t.Fatal(err)
	}

	// Write header
	header := "Time,CellN_m,CellE_m,Elevation_m,PassCount,DesignName,LastAmp,TargPassCount\n"
	f.WriteString(header)

	// Write 100 rows (will test chunking with smaller chunk size)
	for i := 0; i < 100; i++ {
		f.WriteString("2025/Oct/01 09:30:02.800,100,200,5.0,2,Design1,0.97,4\n")
	}
	f.Close()

	// Run sorter
	err = SortCSV(testCSV, outputDir, false, nil)
	if err != nil {
		t.Fatalf("SortCSV with chunking failed: %v", err)
	}

	// Check output exists
	expectedFile := filepath.Join(outputDir, "2025-10-01designDesign1amp097.csv")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Error("Expected output file does not exist")
	}

	// Verify all rows were written
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 101 { // header + 100 rows
		t.Errorf("Expected 101 lines, got %d", len(lines))
	}
}

// TestSortCSVMissingColumns tests error handling for missing columns
func TestSortCSVMissingColumns(t *testing.T) {
	tmpDir := t.TempDir()
	testCSV := filepath.Join(tmpDir, "bad.csv")
	outputDir := filepath.Join(tmpDir, "output")

	// Create CSV without required columns
	csvContent := "Time,CellN_m,CellE_m\n2025/Oct/01 09:30:02.800,100,200\n"
	if err := os.WriteFile(testCSV, []byte(csvContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return error
	err := SortCSV(testCSV, outputDir, false, nil)
	if err == nil {
		t.Error("Expected error for missing columns, got nil")
	}
}
