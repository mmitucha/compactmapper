package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"compactmapper/las"
)

type Point struct {
	X, Y, Z    float64
	R, G, B    uint16
	PassCount  int
	TargetPass int
}

func main() {
	inputDir := flag.String("input", "", "Input directory containing CSV files")
	outputDir := flag.String("output", "", "Output directory for LAS files")
	// skipErrors allows processing to continue when encountering malformed CSV data
	// All errors are logged to err.log in the output directory for later review
	skipErrors := flag.Bool("skip-errors", false, "Skip rows with errors and continue processing (errors logged to err.log)")
	flag.Parse()

	if *inputDir == "" || *outputDir == "" {
		fmt.Println("CompactMapper CLI - CSV to LAS Converter")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  compactmapper-cli -input ./testdata -output ./output")
		fmt.Println("  compactmapper-cli -input ./testdata -output ./output -skip-errors")
		os.Exit(1)
	}

	// Check input directory exists
	if _, err := os.Stat(*inputDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input directory does not exist: %s\n", *inputDir)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Find CSV files
	files, err := filepath.Glob(filepath.Join(*inputDir, "*.csv"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No CSV files found in %s\n", *inputDir)
		os.Exit(0)
	}

	fmt.Printf("Found %d CSV file(s) to convert\n\n", len(files))

	// Setup error logging if skip-errors is enabled
	// This provides an audit trail of all data quality issues encountered during processing
	var errorLog *os.File
	if *skipErrors {
		errorLogPath := filepath.Join(*outputDir, "err.log")
		var err error
		errorLog, err = os.Create(errorLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating error log file: %v\n", err)
			os.Exit(1)
		}
		defer errorLog.Close()
		fmt.Printf("Error logging enabled: %s\n\n", errorLogPath)
	}

	// Convert each file
	successCount := 0
	for i, csvFile := range files {
		fmt.Printf("[%d/%d] Converting %s...", i+1, len(files), filepath.Base(csvFile))

		if err := convertCSVtoLAS(csvFile, *outputDir, *skipErrors, errorLog); err != nil {
			fmt.Printf(" FAILED\n  Error: %v\n", err)
			if errorLog != nil {
				fmt.Fprintf(errorLog, "File: %s - Error: %v\n", filepath.Base(csvFile), err)
			}
			if !*skipErrors {
				continue
			}
		}

		fmt.Printf(" OK\n")
		successCount++
	}

	fmt.Printf("\nConversion complete! %d/%d files successfully converted\n", successCount, len(files))
	if *skipErrors && errorLog != nil {
		fmt.Printf("Errors logged to: %s\n", filepath.Join(*outputDir, "err.log"))
	}
}

// convertCSVtoLAS converts a single CSV file to LAS format
// skipErrors: when true, rows with parsing errors are skipped and logged instead of failing
// errorLog: optional file handle for logging errors (required when skipErrors is true)
// This function is used by the CLI tool; the main converter package is used by GUI and pipeline
func convertCSVtoLAS(csvPath, outputFolder string, skipErrors bool, errorLog *os.File) error {
	// Read CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
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
	required := []string{"CellE_m", "CellN_m", "Elevation_m", "PassCount", "TargPassCount"}
	for _, col := range required {
		if _, ok := colMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	// Parse points
	var points []Point
	skippedRows := 0
	for i := 1; i < len(records); i++ {
		row := records[i]

		x, err := strconv.ParseFloat(row[colMap["CellE_m"]], 64)
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: invalid CellE_m value: %v\n", filepath.Base(csvPath), i+1, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("row %d: invalid CellE_m value: %v", i+1, err)
		}
		y, err := strconv.ParseFloat(row[colMap["CellN_m"]], 64)
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: invalid CellN_m value: %v\n", filepath.Base(csvPath), i+1, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("row %d: invalid CellN_m value: %v", i+1, err)
		}
		z, err := strconv.ParseFloat(row[colMap["Elevation_m"]], 64)
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: invalid Elevation_m value: %v\n", filepath.Base(csvPath), i+1, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("row %d: invalid Elevation_m value: %v", i+1, err)
		}
		passCount, err := strconv.Atoi(row[colMap["PassCount"]])
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: invalid PassCount value: %v\n", filepath.Base(csvPath), i+1, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("row %d: invalid PassCount value: %v", i+1, err)
		}
		targPass, err := strconv.Atoi(row[colMap["TargPassCount"]])
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: invalid TargPassCount value: %v\n", filepath.Base(csvPath), i+1, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("row %d: invalid TargPassCount value: %v", i+1, err)
		}

		// Determine color based on pass count
		var r, g, b uint16
		if passCount < targPass {
			r, g, b = 65535, 0, 0 // Red
		} else if passCount == targPass {
			r, g, b = 0, 65535, 0 // Green
		} else {
			r, g, b = 0, 0, 65535 // Blue
		}

		points = append(points, Point{
			X: x, Y: y, Z: z,
			R: r, G: g, B: b,
			PassCount:  passCount,
			TargetPass: targPass,
		})
	}

	if skipErrors && skippedRows > 0 && errorLog != nil {
		fmt.Fprintf(errorLog, "File: %s - Total skipped rows: %d\n", filepath.Base(csvPath), skippedRows)
	}

	// Create LAS file
	outputPath := filepath.Join(outputFolder, filepath.Base(csvPath[:len(csvPath)-4]+".las"))

	writer := las.NewWriter()

	for _, pt := range points {
		writer.AddPoint(las.Point{
			X:              pt.X,
			Y:              pt.Y,
			Z:              pt.Z,
			R:              pt.R,
			G:              pt.G,
			B:              pt.B,
			Intensity:      0,
			Classification: 1, // Default classification
		})
	}

	return writer.Write(outputPath)
}
