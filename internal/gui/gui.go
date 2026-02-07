package gui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"compactmapper/internal/converter"
	"compactmapper/internal/sorter"
)

// tappableContainer is a container that can be tapped and shows hover effect
type tappableContainer struct {
	widget.BaseWidget
	content    fyne.CanvasObject
	border     *canvas.Rectangle
	onTap      func()
	normalWidth float32
	hoverWidth  float32
}

func newTappableContainer(content fyne.CanvasObject, border *canvas.Rectangle, onTap func()) *tappableContainer {
	t := &tappableContainer{
		content:     content,
		border:      border,
		onTap:       onTap,
		normalWidth: 2,
		hoverWidth:  3,
	}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tappableContainer) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableContainer) MouseIn(*desktop.MouseEvent) {
	t.border.StrokeWidth = t.hoverWidth
	t.border.Refresh()
}

func (t *tappableContainer) MouseOut() {
	t.border.StrokeWidth = t.normalWidth
	t.border.Refresh()
}

func (t *tappableContainer) MouseMoved(*desktop.MouseEvent) {
}

// createSection creates a themed section container with subtle border
func createSection(content fyne.CanvasObject, accentColor color.Color) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = accentColor
	border.StrokeWidth = 2
	border.CornerRadius = 5

	return container.NewStack(border, container.NewPadded(content))
}

// showFileOrFolderDialog shows a dialog to choose between file or folder
func showFileOrFolderDialog(onSelect func(path string, isDir bool), window fyne.Window) {
	fileBtn := widget.NewButton("📄 Select CSV File", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil || uri == nil {
				return
			}
			defer uri.Close()
			onSelect(uri.URI().Path(), false)
		}, window)
	})

	folderBtn := widget.NewButton("📁 Select Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			onSelect(uri.Path(), true)
		}, window)
	})

	content := container.NewVBox(
		widget.NewLabel("Choose input type:"),
		fileBtn,
		folderBtn,
	)

	d := dialog.NewCustom("Select Input", "Cancel", content, window)
	d.Show()
}

// Run starts the GUI application
func Run() {
	myApp := app.New()
	myWindow := myApp.NewWindow("CompactMapper - CAT Roller Data Processor")
	myWindow.Resize(fyne.NewSize(700, 650))

	var inputPath, outputPath string
	var isDirectory bool
	var skipErrors bool

	// Title
	title := widget.NewLabel("CompactMapper")
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := widget.NewLabel("CAT Roller Data Processor")
	subtitle.Alignment = fyne.TextAlignCenter

	// ===== INPUT SECTION =====
	inputHeader := widget.NewLabel("📥 Input")
	inputHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Create input label first
	inputLabel := widget.NewLabel("📁 Drag & drop CSV file or folder here\n\n(or click to browse)")
	inputLabel.Alignment = fyne.TextAlignCenter
	inputLabel.Wrapping = fyne.TextWrapWord
	inputLabel.TextStyle = fyne.TextStyle{Italic: true}

	inputRect := canvas.NewRectangle(color.Transparent)
	inputRect.StrokeColor = theme.PrimaryColor()
	inputRect.StrokeWidth = 2
	inputRect.CornerRadius = 5

	inputContent := container.NewPadded(inputLabel)

	inputTappable := newTappableContainer(container.NewStack(inputRect, inputContent), inputRect, func() {
		showFileOrFolderDialog(func(path string, isDir bool) {
			inputPath = path
			isDirectory = isDir
			if isDir {
				inputLabel.SetText("📁 " + path)
			} else {
				inputLabel.SetText("📄 " + filepath.Base(path) + "\n" + filepath.Dir(path))
			}
			inputLabel.TextStyle = fyne.TextStyle{Bold: true}
			inputLabel.Refresh()
		}, myWindow)
	})

	inputDropZone := inputTappable

	inputSection := createSection(container.NewVBox(
		inputHeader,
		inputDropZone,
	), theme.PrimaryColor())

	// ===== OUTPUT SECTION =====
	outputHeader := widget.NewLabel("📤 Output")
	outputHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Create output label first
	outputLabel := widget.NewLabel("📂 Drag & drop output folder here\n\n(or click to browse)")
	outputLabel.Alignment = fyne.TextAlignCenter
	outputLabel.Wrapping = fyne.TextWrapWord
	outputLabel.TextStyle = fyne.TextStyle{Italic: true}

	outputRect := canvas.NewRectangle(color.Transparent)
	outputRect.StrokeColor = theme.SuccessColor()
	outputRect.StrokeWidth = 2
	outputRect.CornerRadius = 5

	outputContent := container.NewPadded(outputLabel)

	outputTappable := newTappableContainer(container.NewStack(outputRect, outputContent), outputRect, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			path := uri.Path()
			outputPath = path
			outputLabel.SetText("📂 " + path)
			outputLabel.TextStyle = fyne.TextStyle{Bold: true}
			outputLabel.Refresh()
		}, myWindow)
	})

	outputDropZone := outputTappable

	outputInfo := widget.NewLabel("Output structure:\n  📁 output/csv/  - Sorted CSV files\n  📁 output/las/  - LAS point cloud files")

	outputSection := createSection(container.NewVBox(
		outputHeader,
		outputDropZone,
		outputInfo,
	), theme.SuccessColor())

	// ===== OPTIONS SECTION =====
	optionsHeader := widget.NewLabel("⚙️ Options")
	optionsHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Skip errors checkbox - allows processing to continue despite malformed data in source CSV files
	// When enabled, errors are logged to err.log and processing continues
	// This is useful for real-world CAT roller data which may contain sensor glitches or incomplete records
	skipErrorsCheck := widget.NewCheck("Skip errors and continue processing", func(checked bool) {
		skipErrors = checked
	})
	skipErrorsCheck.SetChecked(false)

	skipErrorsInfo := widget.NewLabel("When enabled, errors will be logged to err.log in the output folder")
	skipErrorsInfo.TextStyle = fyne.TextStyle{Italic: true}

	optionsSection := createSection(container.NewVBox(
		optionsHeader,
		skipErrorsCheck,
		skipErrorsInfo,
	), color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// ===== PROCESSING SECTION =====
	statusLabel := widget.NewLabel("Ready to process")
	statusLabel.Alignment = fyne.TextAlignCenter

	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Custom process button with color
	processBtn := widget.NewButton("🚀 Start Processing", func() {
		if inputPath == "" || outputPath == "" {
			dialog.ShowError(fmt.Errorf("please select both input and output paths"), myWindow)
			return
		}

		statusLabel.SetText("Processing...")
		progressBar.Show()
		progressBar.SetValue(0)

		go func() {
			// Setup error logging if skip-errors is enabled
			// This allows the user to review data quality issues after processing completes
			var errorLog *os.File
			if skipErrors {
				errorLogPath := filepath.Join(outputPath, "err.log")
				var err error
				errorLog, err = os.Create(errorLogPath)
				if err != nil {
					progressBar.Hide()
					dialog.ShowError(fmt.Errorf("failed to create error log: %v", err), myWindow)
					statusLabel.SetText("❌ Error creating log file")
					return
				}
				defer errorLog.Close()
				statusLabel.SetText(fmt.Sprintf("Error logging enabled: %s", errorLogPath))
			}

			// Step 1: Sort CSV files
			sortedDir := filepath.Join(outputPath, "csv")
			statusLabel.SetText("Step 1/2: Sorting CSV files...")
			progressBar.SetValue(0.25)

			var err error
			if isDirectory {
				err = sorter.SortCSVDirectory(inputPath, sortedDir, skipErrors, errorLog)
			} else {
				err = sorter.SortCSV(inputPath, sortedDir, skipErrors, errorLog)
			}

			if err != nil && !skipErrors {
				progressBar.Hide()
				dialog.ShowError(fmt.Errorf("sorting failed: %v", err), myWindow)
				statusLabel.SetText("❌ Error during sorting")
				return
			}

			progressBar.SetValue(0.5)

			// Step 2: Convert to LAS
			lasDir := filepath.Join(outputPath, "las")
			statusLabel.SetText("Step 2/2: Converting to LAS...")
			progressBar.SetValue(0.75)

			count, err := converter.ConvertDirectory(sortedDir, lasDir)
			if err != nil {
				progressBar.Hide()
				dialog.ShowError(fmt.Errorf("conversion failed: %v", err), myWindow)
				statusLabel.SetText("❌ Error during conversion")
				return
			}

			progressBar.SetValue(1.0)
			statusLabel.SetText(fmt.Sprintf("✅ Complete! Processed %d files", count))

			successMsg := fmt.Sprintf("Successfully processed data!\n\n"+
				"Sorted CSV files: %s\n"+
				"LAS files: %s\n\n"+
				"Processed %d files",
				sortedDir, lasDir, count)

			if skipErrors && errorLog != nil {
				successMsg += fmt.Sprintf("\n\nErrors logged to: %s", filepath.Join(outputPath, "err.log"))
			}

			dialog.ShowInformation("Success", successMsg, myWindow)

			progressBar.Hide()
		}()
	})
	processBtn.Importance = widget.HighImportance

	// Layout
	content := container.NewVBox(
		container.NewCenter(container.NewVBox(title, subtitle)),
		layout.NewSpacer(),
		inputSection,
		layout.NewSpacer(),
		outputSection,
		layout.NewSpacer(),
		optionsSection,
		layout.NewSpacer(),
		processBtn,
		progressBar,
		statusLabel,
		layout.NewSpacer(),
	)

	myWindow.SetContent(container.NewPadded(content))

	// Handle drag & drop on window
	myWindow.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}

		uri := uris[0]
		path := uri.Path()

		// Check if file or directory
		info, err := os.Stat(path)
		if err != nil {
			return
		}

		isDir := info.IsDir()

		// Determine drop target based on Y position
		windowHeight := myWindow.Canvas().Size().Height
		middleY := windowHeight / 2

		if pos.Y < middleY {
			// Dropped on input area
			inputPath = path
			isDirectory = isDir

			if isDir {
				inputLabel.SetText("📁 " + path)
			} else {
				// Check if it's a CSV file
				if !strings.HasSuffix(strings.ToLower(path), ".csv") {
					dialog.ShowError(fmt.Errorf("please drop a CSV file or folder"), myWindow)
					return
				}
				inputLabel.SetText("📄 " + filepath.Base(path) + "\n" + filepath.Dir(path))
			}
			inputLabel.TextStyle = fyne.TextStyle{Bold: true}
			inputLabel.Refresh()

			// Visual feedback
			inputRect.StrokeWidth = 3
			inputRect.Refresh()
			go func() {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Input Selected",
					Content: filepath.Base(path),
				})
			}()
		} else {
			// Dropped on output area
			if !isDir {
				path = filepath.Dir(path)
			}
			outputPath = path
			outputLabel.SetText("📂 " + path)
			outputLabel.TextStyle = fyne.TextStyle{Bold: true}
			outputLabel.Refresh()

			// Visual feedback
			outputRect.StrokeWidth = 3
			outputRect.Refresh()
			go func() {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Output Selected",
					Content: filepath.Base(path),
				})
			}()
		}
	})

	myWindow.ShowAndRun()
}
