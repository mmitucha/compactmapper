package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"compactmapper/las"
)

type Point struct {
	X, Y, Z    float64
	R, G, B    uint16
	PassCount  int
	TargetPass int
}

// createDropLabel creates a styled label for drop areas
func createDropLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	return label
}

// createDropArea creates a visual drop area with border
func createDropArea(label *widget.Label) *fyne.Container {
	rect := canvas.NewRectangle(color.NRGBA{R: 200, G: 200, B: 200, A: 50})
	return container.NewStack(rect, container.NewPadded(label))
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("CompactMapper - CSV to LAS Converter")
	myWindow.Resize(fyne.NewSize(700, 450))

	var inputFolder, outputFolder string

	// Create drop area labels
	inputLabel := createDropLabel("📁 Drag & drop input folder here\nNot selected")
	outputLabel := createDropLabel("📁 Drag & drop output folder here\nNot selected")

	// Create visual drop areas
	inputDropArea := createDropArea(inputLabel)
	inputDropArea.Resize(fyne.NewSize(650, 60))

	outputDropArea := createDropArea(outputLabel)
	outputDropArea.Resize(fyne.NewSize(650, 60))

	statusLabel := widget.NewLabel("Ready to convert")

	inputBtn := widget.NewButton("Select Input Folder (CSV files)", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			inputFolder = uri.Path()
			inputLabel.SetText("📁 Input: " + inputFolder)
		}, myWindow)
	})

	outputBtn := widget.NewButton("Select Output Folder (LAS files)", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputFolder = uri.Path()
			outputLabel.SetText("📁 Output: " + outputFolder)
		}, myWindow)
	})

	convertBtn := widget.NewButton("Convert CSV to LAS", func() {
		if inputFolder == "" || outputFolder == "" {
			dialog.ShowError(fmt.Errorf("please select both input and output folders"), myWindow)
			return
		}

		statusLabel.SetText("Converting...")
		go func() {
			count, err := convertFiles(inputFolder, outputFolder, statusLabel)
			if err != nil {
				dialog.ShowError(err, myWindow)
				statusLabel.SetText("Error occurred")
			} else {
				statusLabel.SetText(fmt.Sprintf("Conversion complete! %d files processed 🎉", count))
				dialog.ShowInformation("Success", fmt.Sprintf("Successfully converted %d CSV files to LAS!", count), myWindow)
			}
		}()
	})

	content := container.NewVBox(
		widget.NewLabel("CompactMapper - Point Cloud Converter"),
		widget.NewSeparator(),
		widget.NewLabel("Input Folder (CSV files):"),
		inputDropArea,
		inputBtn,
		widget.NewSeparator(),
		widget.NewLabel("Output Folder (LAS files):"),
		outputDropArea,
		outputBtn,
		widget.NewSeparator(),
		convertBtn,
		statusLabel,
	)

	myWindow.SetContent(content)

	// Set up window-wide drag & drop handler (Fyne 2.4+)
	myWindow.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}

		// Get the first dropped item
		uri := uris[0]
		path := uri.Path()

		// Check if it's a directory
		info, err := os.Stat(path)
		if err != nil {
			return
		}

		// Use parent directory if a file was dropped
		if !info.IsDir() {
			path = filepath.Dir(path)
		}

		// Determine which drop area based on Y position
		// Input area is in the upper half, output in the lower half
		if pos.Y < float32(myWindow.Canvas().Size().Height)/2 {
			// Dropped on input area
			inputFolder = path
			inputLabel.SetText("📁 Input: " + path)
		} else {
			// Dropped on output area
			outputFolder = path
			outputLabel.SetText("📁 Output: " + path)
		}
	})

	myWindow.ShowAndRun()
}

func convertFiles(inputFolder, outputFolder string, statusLabel *widget.Label) (int, error) {
	files, err := filepath.Glob(filepath.Join(inputFolder, "*.csv"))
	if err != nil {
		return 0, err
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no CSV files found in input folder")
	}

	for i, filePath := range files {
		statusLabel.SetText(fmt.Sprintf("Processing %d/%d: %s", i+1, len(files), filepath.Base(filePath)))

		if err := convertCSVtoLAS(filePath, outputFolder); err != nil {
			return i, fmt.Errorf("error processing %s: %v", filepath.Base(filePath), err)
		}
	}

	return len(files), nil
}

func convertCSVtoLAS(csvPath, outputFolder string) error {
	// Read CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() // Read-only file; close error is non-actionable

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
	for i := 1; i < len(records); i++ {
		row := records[i]

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

	// Create LAS file
	outputPath := filepath.Join(outputFolder, filepath.Base(csvPath[:len(csvPath)-4]+".las"))

	writer := las.NewWriter()

	for _, pt := range points {
		writer.AddPoint(las.Point{
			X:     pt.X,
			Y:     pt.Y,
			Z:     pt.Z,
			R:     pt.R,
			G:     pt.G,
			B:     pt.B,
			Intensity: 0,
			Classification: 1, // Default classification
		})
	}

	return writer.Write(outputPath)
}
