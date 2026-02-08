package converter

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"compactmapper/las"
)

// determineColor assigns RGB color based on PassCount vs TargPassCount
// Red: PassCount < TargPassCount (under target)
// Green: PassCount == TargPassCount (at target)
// Blue: PassCount > TargPassCount (over target)
func determineColor(passCount, targPass int) (uint16, uint16, uint16) {
	if passCount < targPass {
		return 65535, 0, 0 // Red
	} else if passCount == targPass {
		return 0, 65535, 0 // Green
	}
	return 0, 0, 65535 // Blue
}

// parseTimeToGPS converts CSV time string to GPS time (seconds since Unix epoch)
// Expected format: "2025/Oct/01 09:31:49.500"
func parseTimeToGPS(timeStr string) (float64, error) {
	// Try parsing with milliseconds
	t, err := time.Parse("2006/Jan/02 15:04:05.000", timeStr)
	if err != nil {
		// Try without milliseconds
		t, err = time.Parse("2006/Jan/02 15:04:05", timeStr)
		if err != nil {
			return 0, fmt.Errorf("invalid time format: %v", err)
		}
	}

	// Convert to GPS time (Adjusted Standard GPS Time = Unix timestamp)
	// This is seconds since January 1, 1970 00:00:00 UTC
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9, nil
}

// ConvertCSVToLAS converts a single CSV file to LAS format
func ConvertCSVToLAS(csvPath, outputDir string) error {
	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("error opening CSV file: %v", err)
	}
	defer func() { _ = file.Close() }() // Read-only file; close error is non-actionable

	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %v", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data rows")
	}

	// Parse header
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Check required columns
	required := []string{"Time", "CellE_m", "CellN_m", "Elevation_m", "PassCount", "TargPassCount"}
	for _, col := range required {
		if _, ok := colMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Create LAS writer
	writer := las.NewWriter()

	// Parse points
	for i := 1; i < len(records); i++ {
		row := records[i]

		// Parse time
		gpsTime, err := parseTimeToGPS(row[colMap["Time"]])
		if err != nil {
			return fmt.Errorf("row %d: invalid Time value: %v", i+1, err)
		}

		x, err := strconv.ParseFloat(row[colMap["CellE_m"]], 64)
		if err != nil {
			return fmt.Errorf("row %d: invalid CellE_m value: %v", i+1, err)
		}

		y, err := strconv.ParseFloat(row[colMap["CellN_m"]], 64)
		if err != nil {
			return fmt.Errorf("row %d: invalid CellN_m value: %v", i+1, err)
		}

		z, err := strconv.ParseFloat(row[colMap["Elevation_m"]], 64)
		if err != nil {
			return fmt.Errorf("row %d: invalid Elevation_m value: %v", i+1, err)
		}

		passCount, err := strconv.Atoi(row[colMap["PassCount"]])
		if err != nil {
			return fmt.Errorf("row %d: invalid PassCount value: %v", i+1, err)
		}

		targPass, err := strconv.Atoi(row[colMap["TargPassCount"]])
		if err != nil {
			return fmt.Errorf("row %d: invalid TargPassCount value: %v", i+1, err)
		}

		// Determine color based on pass count
		r, g, b := determineColor(passCount, targPass)

		// Add point to writer
		writer.AddPoint(las.Point{
			X:              x,
			Y:              y,
			Z:              z,
			R:              r,
			G:              g,
			B:              b,
			Intensity:      0,
			Classification: 1,
			GPSTime:        gpsTime,
		})
	}

	// Generate output filename
	baseName := filepath.Base(csvPath)
	lasName := baseName[:len(baseName)-4] + ".las"
	outputPath := filepath.Join(outputDir, lasName)

	// Write LAS file
	if err := writer.Write(outputPath); err != nil {
		return fmt.Errorf("error writing LAS file: %v", err)
	}

	return nil
}

// ConvertDirectory converts all CSV files in a directory to LAS format
func ConvertDirectory(inputDir, outputDir string) (int, error) {
	// Find all CSV files
	files, err := filepath.Glob(filepath.Join(inputDir, "*.csv"))
	if err != nil {
		return 0, fmt.Errorf("error scanning directory: %v", err)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no CSV files found in %s", inputDir)
	}

	// Convert each file
	successCount := 0
	for _, csvFile := range files {
		if err := ConvertCSVToLAS(csvFile, outputDir); err != nil {
			return successCount, fmt.Errorf("error converting %s: %v", filepath.Base(csvFile), err)
		}
		successCount++
	}

	return successCount, nil
}
