package sorter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// ChunkSize defines how many rows to process at once (matches Python version)
	ChunkSize = 10000
)

// sanitizeFilename removes invalid filename characters
func sanitizeFilename(filename string) string {
	// Remove characters not allowed in filenames: < > : " / \ | ? *
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	return re.ReplaceAllString(filename, "")
}

// parseDate extracts date from Time field format: "2025/Oct/01 09:30:02.800"
func parseDate(timeStr string) (string, error) {
	// Parse format: "2025/Oct/01 09:30:02.800"
	t, err := time.Parse("2006/Jan/02 15:04:05.999", timeStr)
	if err != nil {
		return "", fmt.Errorf("invalid time format: %v", err)
	}
	// Return as "2025-10-01"
	return t.Format("2006-01-02"), nil
}

// normalizeAmp normalizes amplitude value (e.g., "0.97" -> "097", "2.10" -> "210")
func normalizeAmp(amp string) string {
	// Handle empty or missing values
	if amp == "" || amp == "?" {
		return "no_amp"
	}

	// Remove decimal point and take first 3 characters
	normalized := strings.ReplaceAll(amp, ".", "")
	if len(normalized) > 3 {
		normalized = normalized[:3]
	}

	return normalized
}

// generateFilename creates filename from date, design, and amp
func generateFilename(date, design, amp string) string {
	// Format: {date}design{design}amp{amp}.csv
	// Handle "no_amp" case - just use "amp.csv" instead of "ampno_amp.csv"
	if amp == "no_amp" {
		return fmt.Sprintf("%sdesign%samp.csv", date, design)
	}
	return fmt.Sprintf("%sdesign%samp%s.csv", date, design, amp)
}

// GroupKey represents a unique combination of Date, DesignName, and LastAmp
type GroupKey struct {
	Date       string
	DesignName string
	Amp        string
}

// SortCSV processes a single CSV file and groups it by Date, DesignName, and LastAmp
// skipErrors: when true, rows with parsing errors are skipped and logged instead of stopping execution
// errorLog: optional file handle for logging errors (required when skipErrors is true)
// This allows processing to continue even when source CSV files contain corrupt or malformed data
func SortCSV(inputPath, outputDir string, skipErrors bool, errorLog *os.File) error {
	// Open input file
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Read all content to handle BOM
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	// Remove UTF-8 BOM if present (EF BB BF)
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(content))

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("error reading header: %v", err)
	}

	// Find column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Validate required columns
	required := []string{"Time", "DesignName", "LastAmp"}
	for _, col := range required {
		if _, ok := colMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	timeIdx := colMap["Time"]
	designIdx := colMap["DesignName"]
	ampIdx := colMap["LastAmp"]

	// Process in chunks
	groups := make(map[GroupKey][][]string)
	chunk := make([][]string, 0, ChunkSize)
	rowCount := 0

	// Track skipped rows to provide feedback on data quality issues
	skippedRows := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			// Process last chunk
			if len(chunk) > 0 {
				skipped, err := processChunk(chunk, timeIdx, designIdx, ampIdx, groups, skipErrors, errorLog, filepath.Base(inputPath))
				skippedRows += skipped
				if err != nil && !skipErrors {
					return err
				}
			}
			break
		}
		if err != nil {
			// If skip-errors is enabled, log the error and continue processing
			// This prevents a single malformed row from stopping the entire pipeline
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row %d: error reading row: %v\n", filepath.Base(inputPath), rowCount+2, err)
				}
				skippedRows++
				continue
			}
			return fmt.Errorf("error reading row %d: %v", rowCount+2, err)
		}

		chunk = append(chunk, record)
		rowCount++

		// Process chunk when it reaches ChunkSize
		if len(chunk) >= ChunkSize {
			skipped, err := processChunk(chunk, timeIdx, designIdx, ampIdx, groups, skipErrors, errorLog, filepath.Base(inputPath))
			skippedRows += skipped
			if err != nil && !skipErrors {
				return err
			}
			chunk = make([][]string, 0, ChunkSize)
		}
	}

	if skipErrors && skippedRows > 0 && errorLog != nil {
		fmt.Fprintf(errorLog, "File: %s - Total skipped rows during sorting: %d\n", filepath.Base(inputPath), skippedRows)
	}

	// Write grouped data to files
	for key, rows := range groups {
		filename := generateFilename(key.Date, key.DesignName, key.Amp)
		sanitized := sanitizeFilename(filename)
		outputPath := filepath.Join(outputDir, sanitized)

		// Check if file exists (append mode for chunked processing)
		fileExists := false
		if _, err := os.Stat(outputPath); err == nil {
			fileExists = true
		}

		outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("error creating output file %s: %v", sanitized, err)
		}

		writer := csv.NewWriter(outFile)

		// Write header only if file is new
		if !fileExists {
			if err := writer.Write(header); err != nil {
				outFile.Close()
				return fmt.Errorf("error writing header to %s: %v", sanitized, err)
			}
		}

		// Write data rows
		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				outFile.Close()
				return fmt.Errorf("error writing row to %s: %v", sanitized, err)
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			outFile.Close()
			return fmt.Errorf("error flushing writer for %s: %v", sanitized, err)
		}

		outFile.Close()
	}

	return nil
}

// processChunk processes a chunk of rows and groups them
// Returns the number of skipped rows and any fatal error
// When skipErrors is true, parsing errors are logged and the row is skipped
func processChunk(chunk [][]string, timeIdx, designIdx, ampIdx int, groups map[GroupKey][][]string, skipErrors bool, errorLog *os.File, filename string) (int, error) {
	skippedRows := 0
	for i, row := range chunk {
		// Parse date from Time column
		date, err := parseDate(row[timeIdx])
		if err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "File: %s, Row in chunk %d: error parsing date from '%s': %v\n", filename, i+1, row[timeIdx], err)
				}
				skippedRows++
				continue
			}
			return skippedRows, fmt.Errorf("error parsing date from '%s': %v", row[timeIdx], err)
		}

		design := row[designIdx]
		amp := normalizeAmp(row[ampIdx])

		key := GroupKey{
			Date:       date,
			DesignName: design,
			Amp:        amp,
		}

		groups[key] = append(groups[key], row)
	}
	return skippedRows, nil
}

// SortCSVDirectory processes all CSV files in a directory
// skipErrors: when true, files with errors are logged and processing continues with remaining files
// errorLog: optional file handle for logging errors (required when skipErrors is true)
func SortCSVDirectory(inputDir, outputDir string, skipErrors bool, errorLog *os.File) error {
	// Find all CSV files in input directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.csv"))
	if err != nil {
		return fmt.Errorf("error scanning directory: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no CSV files found in %s", inputDir)
	}

	// Process each file
	for _, file := range files {
		if err := SortCSV(file, outputDir, skipErrors, errorLog); err != nil {
			if skipErrors {
				if errorLog != nil {
					fmt.Fprintf(errorLog, "Error processing %s: %v\n", filepath.Base(file), err)
				}
				continue
			}
			return fmt.Errorf("error processing %s: %v", filepath.Base(file), err)
		}
	}

	return nil
}
