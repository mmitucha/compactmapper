package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"compactmapper/internal/converter"
	"compactmapper/internal/gui"
	"compactmapper/internal/sorter"
)

// version is set via ldflags during build (see Makefile)
var version = "dev"

func main() {
	// Define CLI flags
	inputFlag := flag.String("input", "", "Input CSV file or directory")
	outputFlag := flag.String("output", "", "Output directory")
	sortOnlyFlag := flag.Bool("sort-only", false, "Only sort CSV files (skip LAS conversion)")
	convertOnlyFlag := flag.Bool("convert-only", false, "Only convert CSV to LAS (assume already sorted)")
	skipErrorsFlag := flag.Bool("skip-errors", false, "Skip rows with errors and continue processing (errors logged to err.log)")
	versionFlag := flag.Bool("version", false, "Show version information")
	guiFlag := flag.Bool("gui", false, "Launch GUI (default if no flags provided)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CompactMapper v%s - CAT Roller Data Processor\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  compactmapper [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  GUI Mode (default):\n")
		fmt.Fprintf(os.Stderr, "    compactmapper              Launch graphical interface\n")
		fmt.Fprintf(os.Stderr, "    compactmapper --gui        Explicitly launch GUI\n\n")
		fmt.Fprintf(os.Stderr, "  CLI Mode:\n")
		fmt.Fprintf(os.Stderr, "    compactmapper --input <path> --output <dir>\n")
		fmt.Fprintf(os.Stderr, "                            Process CSV files (sort + convert to LAS)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Launch GUI\n")
		fmt.Fprintf(os.Stderr, "  compactmapper\n\n")
		fmt.Fprintf(os.Stderr, "  # Process a single CSV file\n")
		fmt.Fprintf(os.Stderr, "  compactmapper --input data.csv --output ./results\n\n")
		fmt.Fprintf(os.Stderr, "  # Process all CSV files in a directory\n")
		fmt.Fprintf(os.Stderr, "  compactmapper --input ./csvdata --output ./results\n\n")
		fmt.Fprintf(os.Stderr, "  # Only sort CSV files\n")
		fmt.Fprintf(os.Stderr, "  compactmapper --input data.csv --output ./sorted --sort-only\n\n")
		fmt.Fprintf(os.Stderr, "  # Only convert sorted CSVs to LAS\n")
		fmt.Fprintf(os.Stderr, "  compactmapper --input ./sorted --output ./las --convert-only\n\n")
		fmt.Fprintf(os.Stderr, "  # Skip errors and log them\n")
		fmt.Fprintf(os.Stderr, "  compactmapper --input ./csvdata --output ./results --skip-errors\n\n")
		fmt.Fprintf(os.Stderr, "Output Structure:\n")
		fmt.Fprintf(os.Stderr, "  output/\n")
		fmt.Fprintf(os.Stderr, "    csv/  - Sorted CSV files grouped by Date/Design/Amplitude\n")
		fmt.Fprintf(os.Stderr, "    las/  - LAS point cloud files with color-coded pass counts\n")
		fmt.Fprintf(os.Stderr, "\nColor Coding (LAS files):\n")
		fmt.Fprintf(os.Stderr, "  Red:   PassCount < TargetPassCount (under target)\n")
		fmt.Fprintf(os.Stderr, "  Green: PassCount == TargetPassCount (at target)\n")
		fmt.Fprintf(os.Stderr, "  Blue:  PassCount > TargetPassCount (over target)\n")
	}

	flag.Parse()

	// Show version
	if *versionFlag {
		fmt.Printf("CompactMapper v%s\n", version)
		os.Exit(0)
	}

	// Determine mode: GUI or CLI
	// Launch GUI if: explicitly requested OR no arguments provided
	if *guiFlag || (flag.NFlag() == 0 && len(os.Args) == 1) {
		gui.Run()
		return
	}

	// CLI mode - validate required flags
	if *inputFlag == "" || *outputFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --input and --output are required for CLI mode\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate input exists
	inputInfo, err := os.Stat(*inputFlag)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input path does not exist: %s\n", *inputFlag)
		os.Exit(1)
	}

	isDirectory := inputInfo.IsDir()

	// Determine what to do
	if *convertOnlyFlag {
		// Only convert CSV to LAS
		runConvertOnly(*inputFlag, *outputFlag)
	} else if *sortOnlyFlag {
		// Only sort CSV
		runSortOnly(*inputFlag, *outputFlag, isDirectory, *skipErrorsFlag)
	} else {
		// Full pipeline: sort + convert
		runFullPipeline(*inputFlag, *outputFlag, isDirectory, *skipErrorsFlag)
	}
}

func runSortOnly(input, output string, isDirectory bool, skipErrors bool) {
	fmt.Println("Starting CSV sorting...")

	// Setup error logging if skip-errors is enabled
	var errorLog *os.File
	if skipErrors {
		errorLogPath := filepath.Join(output, "err.log")
		var err error
		errorLog, err = os.Create(errorLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating error log file: %v\n", err)
			os.Exit(1)
		}
		defer errorLog.Close()
		fmt.Printf("Error logging enabled: %s\n", errorLogPath)
	}

	var err error
	if isDirectory {
		fmt.Printf("Processing directory: %s\n", input)
		err = sorter.SortCSVDirectory(input, output, skipErrors, errorLog)
	} else {
		fmt.Printf("Processing file: %s\n", input)
		err = sorter.SortCSV(input, output, skipErrors, errorLog)
	}

	if err != nil && !skipErrors {
		fmt.Fprintf(os.Stderr, "Error during sorting: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Sorting complete!\n")
	fmt.Printf("  Output: %s\n", output)
	if skipErrors && errorLog != nil {
		fmt.Printf("  Errors logged to: %s\n", filepath.Join(output, "err.log"))
	}
}

func runConvertOnly(input, output string) {
	fmt.Println("Starting CSV to LAS conversion...")
	fmt.Printf("Input directory: %s\n", input)

	count, err := converter.ConvertDirectory(input, output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during conversion: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Conversion complete!\n")
	fmt.Printf("  Processed: %d files\n", count)
	fmt.Printf("  Output: %s\n", output)
}

func runFullPipeline(input, output string, isDirectory bool, skipErrors bool) {
	// Setup error logging if skip-errors is enabled
	var errorLog *os.File
	if skipErrors {
		errorLogPath := filepath.Join(output, "err.log")
		var err error
		errorLog, err = os.Create(errorLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating error log file: %v\n", err)
			os.Exit(1)
		}
		defer errorLog.Close()
		fmt.Printf("Error logging enabled: %s\n\n", errorLogPath)
	}

	// Step 1: Sort CSV files
	sortedDir := filepath.Join(output, "csv")
	fmt.Println("Step 1/2: Sorting CSV files...")

	var err error
	if isDirectory {
		fmt.Printf("Processing directory: %s\n", input)
		err = sorter.SortCSVDirectory(input, sortedDir, skipErrors, errorLog)
	} else {
		fmt.Printf("Processing file: %s\n", input)
		err = sorter.SortCSV(input, sortedDir, skipErrors, errorLog)
	}

	if err != nil && !skipErrors {
		fmt.Fprintf(os.Stderr, "Error during sorting: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Sorted CSV files: %s\n\n", sortedDir)

	// Step 2: Convert to LAS
	lasDir := filepath.Join(output, "las")
	fmt.Println("Step 2/2: Converting to LAS...")

	count, err := converter.ConvertDirectory(sortedDir, lasDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during conversion: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🎉 Processing complete!\n")
	fmt.Printf("  Sorted CSV: %s\n", sortedDir)
	fmt.Printf("  LAS files:  %s\n", lasDir)
	fmt.Printf("  Total:      %d files\n", count)
	if skipErrors && errorLog != nil {
		fmt.Printf("  Errors logged to: %s\n", filepath.Join(output, "err.log"))
	}
}
