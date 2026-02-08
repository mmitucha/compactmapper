# CompactMapper

**CompactMapper** is a purpose-built cross-platform desktop application for sorting and processing roller compaction CSV data from CAT Compaction Control / MDP systems.

**Primary Purpose:** Sort Intelligent Compaction (IC) data by specific segmentation criteria (Date, Design/Task, Machine Settings) to organize multi-machine, multi-task datasets into manageable files.

**Secondary Feature:** Convert sorted CSV files to LAS point cloud format with RGB color-coded pass-count visualization.

> [!NOTE]
> **🎵 Origin & Development**
>
> Built to solve **one specific problem**: processing CAT Intelligent Compaction data from multiple rollers across construction tasks. Originally a Python script, **vibecoded** with [Claude Code](https://claude.ai/code) into a Go + Fyne GUI application for **single-click accessibility**. Single statically-linked executable works on Mac/Linux/Windows, making accessible to non-technical audience.

## Features

- **User-Friendly GUI**: Drag-and-drop interface - no command line needed, no dependencies to install
  - **Perfect for non-technical users**: Field engineers and operators can process data with a single click
  - **Single executable**: Download once, run anywhere (Mac/Linux/Windows)

- **Primary: Intelligent CSV Sorting** - Automatically segments compaction data by configurable criteria:
  - **Date**: YYYY-MM-DD format (parsed from Time column) - groups by operation day
  - **Design/Task**: From DesignName column - separates different construction zones
  - **Machine Settings**: From LastAmp column (normalized) - distinguishes operational modes

  *This is the core feature - organizing multi-machine, multi-task data into manageable segmented files.*

- **Secondary: LAS Conversion** - Creates point cloud files with RGB color coding based on pass counts
  - 🔴 Red: Under target passes
  - 🟢 Green: At target passes
  - 🔵 Blue: Over target passes

- **Dual Interface**: GUI (default) for non-technical users, CLI for automation/scripting
- **Cross-Platform**: Single static binary works on Windows, Linux, and macOS
- **Fast Processing**: Handles large files efficiently with chunked processing

## Quick Start

### GUI Mode (Default)

```bash
./compactmapper
```

### CLI Mode

```bash
# Process a single CSV file
./compactmapper --input data.csv --output ./results

# Process all CSV files in a directory
./compactmapper --input ./csvdata --output ./results
```

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/yourusername/compactmapper/releases) page.

### Build from Source

**Requirements:**
- Go 1.21 or later
- CGO-enabled compiler (for GUI support)

```bash
# Clone the repository
git clone https://github.com/yourusername/compactmapper.git
cd compactmapper

# Build
make build

# Or build manually
CGO_ENABLED=1 go build -o compactmapper ./cmd/compactmapper
```

## Usage

### GUI Mode

Simply run the binary without arguments to launch the graphical interface:

```bash
./compactmapper
# or explicitly
./compactmapper --gui
```

**GUI Workflow:**
1. Select input (CSV file or directory)
2. Select output directory
3. Click "Process Data"
4. Results will be in:
   - `output/csv/` - Sorted CSV files
   - `output/las/` - LAS point cloud files

### CLI Mode

**Full Pipeline (Sort + Convert):**

```bash
./compactmapper --input data.csv --output ./results
```

**Sort Only:**

```bash
./compactmapper --input data.csv --output ./sorted --sort-only
```

**Convert Only (from already sorted CSVs):**

```bash
./compactmapper --input ./sorted --output ./las --convert-only
```

**CLI Options:**

```
  -input string
      Input CSV file or directory
  -output string
      Output directory
  -sort-only
      Only sort CSV files (skip LAS conversion)
  -convert-only
      Only convert CSV to LAS (assume already sorted)
  -version
      Show version information
  -gui
      Launch GUI (default if no flags provided)
```

## Output Structure

```
output/
├── csv/                                      # Sorted CSV files
│   ├── 2025-10-01designDesign1amp097.csv
│   ├── 2025-10-01designDesign2amp210.csv
│   └── ...
└── las/                                      # LAS point cloud files
    ├── 2025-10-01designDesign1amp097.las
    ├── 2025-10-01designDesign2amp210.las
    └── ...
```

### File Naming Convention

Sorted files are named: `{Date}design{DesignName}amp{Amplitude}.csv`

- **Date**: YYYY-MM-DD format (parsed from Time column)
- **DesignName**: From CSV DesignName column
- **Amplitude**: Normalized from LastAmp column (e.g., "0.97" → "097")

## Data Processing

### CSV Sorting (Primary Feature)

**Purpose:** Organize Intelligent Compaction (IC) data from multiple rollers and tasks into separate files based on segmentation criteria.

Input CSV files are grouped by three key criteria:
1. **Date** - Extracted from Time field (format: `2025/Oct/01 09:30:02.800`)
   - Groups data by day of operation
2. **Design/Task** - From DesignName column
   - Separates different construction tasks or design zones
3. **Machine Settings** - From LastAmp (amplitude) column, normalized
   - Distinguishes different operational modes or machine configurations

**Required CSV Columns for Sorting:**
- `Time` - Timestamp (format: `YYYY/MMM/DD HH:MM:SS.mmm`)
- `DesignName` - Design/task identifier
- `LastAmp` - Amplitude/machine setting value

**Use Case:** Compaction data from CAT Compaction Control / MDP systems (IC data recorded during rolling) gathered from multiple machines across different tasks can be easily segmented using these criteria.

### LAS Conversion (Secondary Feature)

Creates LAS 1.2 Point Format 3 (with RGB and GPS Time) files from sorted CSV data.

**Required CSV Columns for Conversion:**
- `CellE_m` - Easting coordinate (X)
- `CellN_m` - Northing coordinate (Y)
- `Elevation_m` - Elevation (Z)
- `PassCount` - Number of passes
- `TargPassCount` - Target pass count

**Color Assignment:**
- PassCount < TargPassCount → **Red (65535, 0, 0)** - Under target
- PassCount == TargPassCount → **Green (0, 65535, 0)** - At target
- PassCount > TargPassCount → **Blue (0, 0, 65535)** - Over target

## Development

### Project Structure

```
compactmapper/
├── cmd/
│   └── compactmapper/          # Main entry point (GUI + CLI)
│       └── main.go
├── internal/
│   ├── sorter/              # CSV sorting logic
│   │   ├── sorter.go
│   │   └── sorter_test.go
│   ├── converter/           # CSV to LAS conversion
│   │   ├── converter.go
│   │   └── converter_test.go
│   └── gui/                 # GUI implementation
│       └── gui.go
├── las/                     # LAS file reader/writer
│   ├── writer.go
│   ├── reader.go
│   └── writer_test.go
├── test/                    # Integration tests
│   └── integration_test.go
├── sample/                  # Test data
│   ├── 0_src/              # Source CSV data
│   ├── 1_sorted/           # Expected sorted output
│   └── 2_las/              # Expected LAS output
├── go.mod
├── Makefile
└── README.md
```

### Building & Testing

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run integration tests
make test-integration

# Build for current platform (includes OS/arch in filename)
make build

# Build for specific platforms
make build-macos    # macOS binary
make build-windows  # Windows binary (requires mingw-w64)

# Test with sample data
make test-sample

# Clean build artifacts
make clean

# Show all available targets
make help
```

### Running Tests

```bash
# All tests (unit + integration)
go test ./...

# Unit tests only
go test ./internal/sorter ./internal/converter ./las

# Integration tests with sample data
go test -v ./test

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Cross-Platform Building

### macOS

```bash
# Native build (includes OS/arch in filename)
make build
# Output: build/compactmapper-v0.2.0-darwin-arm64 (or darwin-amd64)
# Symlink: build/compactmapper -> compactmapper-v0.2.0-darwin-arm64

# Platform-specific build
make build-macos
# Output: build/compactmapper-v0.2.0-darwin-amd64
```

### Linux

**Cross-compilation from macOS is not supported** due to Fyne GUI requirements (X11, OpenGL headers).

**Options:**
1. **Native build on Linux:**
   ```bash
   make build
   # Output: build/compactmapper-v0.2.0-linux-amd64
   ```

2. **GitHub Actions:** Linux binaries are built automatically in CI/CD pipeline

### Windows

Requires MinGW-w64 cross-compiler with CGO optimization flags:

```bash
# Install mingw-w64 (macOS)
brew install mingw-w64

# Build (optimized to prevent hanging)
make build-windows
# Output: build/compactmapper-v0.2.0-windows-amd64.exe
```

**Note:** Windows cross-compilation uses `-O2` CGO optimization flags to prevent compilation hangs with Fyne's OpenGL bindings.
```

## Technical Details

### LAS File Format

- **Version**: LAS 1.2
- **Point Format**: 2 (includes RGB)
- **Point Record Length**: 26 bytes
- **Scale Factor**: 0.001 (for improved precision)
- **Coordinate System**: Preserved from input CSV

### CSV Processing

- **Chunk Size**: 10,000 rows (configurable in code)
- **BOM Handling**: Automatically strips UTF-8 BOM
- **Memory Efficient**: Streams large files
- **Filename Sanitization**: Removes invalid characters

### Dependencies

- [Fyne](https://fyne.io/) v2.4.5 - Cross-platform GUI framework
- Go standard library - CSV parsing, file I/O

## Contributing

Contributions are welcome! Please follow these guidelines:

1. **Fork** the repository
2. Create a **feature branch** (`git checkout -b feature/amazing-feature`)
3. **Write tests** for new functionality
4. Ensure all tests pass (`make test`)
5. **Format code** (`make fmt`)
6. **Commit** changes (`git commit -m 'Add amazing feature'`)
7. **Push** to branch (`git push origin feature/amazing-feature`)
8. Open a **Pull Request**

### Code Style

- Follow standard Go conventions
- Use `go fmt` for formatting
- Write descriptive comments for exported functions
- Include tests for new features
- Test-Driven Development (TDD) approach preferred

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original Python implementation by Tomas
- LAS format specification: [ASPRS LAS 1.2](https://www.asprs.org/wp-content/uploads/2010/12/asprs_las_format_v12.pdf)
- Built with [Fyne](https://fyne.io/) GUI toolkit

## Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/yourusername/compactmapper/issues)
- 📖 **Documentation**: This README and code comments
- 💬 **Discussions**: [GitHub Discussions](https://github.com/yourusername/compactmapper/discussions)

---

**CompactMapper** - Efficiently process and visualize CAT roller compaction data
