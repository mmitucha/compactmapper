# CompactMapper - Claude Code Instructions

## Overview
**Purpose-built tool** for sorting roller compaction CSV data by specific segmentation criteria, then optionally converting to LAS point clouds.

**Primary Purpose:** Sort Intelligent Compaction (IC) CSV data by three key criteria:
- **Date**: YYYY-MM-DD (from Time column) - groups by operation day
- **Design/Task**: From DesignName column - separates construction zones
- **Machine Settings**: From LastAmp column (normalized) - distinguishes operational modes

This sorting is the **core feature** - organizing multi-machine, multi-task datasets into manageable segmented files.

**Secondary Feature:** Convert sorted CSV → LAS 1.2 Format 3 point clouds with RGB pass-count visualization (Red=under, Green=optimal, Blue=over target) and GPS time preservation.

**Stack:** Go 1.21+ | Fyne v2.4.5 (GUI) | Makefile | CSV→LAS 1.2 Format 3

**Origin:** Built for CAT Compaction Control / MDP systems IC data from multiple rollers. Python → Go + Fyne for **single-click GUI** (no CLI/dependencies). Major driver: field engineers shouldn't need technical setup.

## Structure
```
cmd/compactmapper/main.go       # Entry point (GUI + CLI unified)
internal/sorter/             # CSV grouping by Date/DesignName/LastAmp
internal/converter/          # CSV→LAS with RGB coloring
internal/gui/                # Fyne GUI with drag&drop
las/                         # Custom LAS 1.2 writer/reader
test/                        # Integration test code
testdata/integration/        # Integration test fixtures (input/, expected_sorted/, expected_las/)
tools/data/                  # Data utilities (anonymize_data.py, samplify_data.py)
```
> `main.go` in root and `cmd/cli/` are deprecated.

## Build & Test
```bash
make build              # Current platform (CGO required for Fyne)
make build-macos        # macOS (darwin-amd64)
make build-windows      # Windows (requires mingw-w64)
make test-all           # Validate: all test levels (unit + integration + e2e)
make clean / fmt / lint
```

**Outputs:** `build/compactmapper-v{VERSION}-{GOOS}-{GOARCH}[.exe]` + symlink `build/compactmapper`

**Linux:** Build natively (cross-compilation unsupported) or use GitHub Actions CI.

## CLI
**Flags:** `--input <path>` `--output <dir>` `--sort-only` `--convert-only` `--skip-errors` `--version` `--gui`

**CSV Columns:**
- Sorting: `Time`, `DesignName`, `LastAmp`
- Conversion: `Time`, `CellE_m`, `CellN_m`, `Elevation_m`, `PassCount`, `TargPassCount`

## Components
- **Sorter** (`internal/sorter`): Groups by Date/DesignName/LastAmp, 10K chunks, UTF-8 BOM handling
- **Converter** (`internal/converter`): RGB from PassCount vs TargPassCount (R=under, G=at, B=over)
- **LAS Writer** (`las/writer.go`): LAS 1.2 Format 3, 34 bytes/point (XYZ+Intensity+GPS+RGB), 0.001 scale
- **GUI** (`internal/gui`): Fyne drag&drop, progress tracking

**Output:** `{YYYY-MM-DD}design{DesignName}amp{NormalizedAmp}.csv|.las`

## Testing
**Validate all changes:** `make test-all` (runs unit + integration + e2e)

**Test Levels:**
1. Unit: `internal/sorter`, `internal/converter`, `las` (inline test data)
2. Integration: `test/integration_test.go` (uses `testdata/integration/`)
3. E2E: `test/integration_test.sh` (black-box binary testing)

## Conventions
- `go fmt`, error wrapping: `fmt.Errorf("ctx: %w", err)`, doc comments on exports
- Version: `go build -ldflags "-X main.version=v1.2.0" ./cmd/compactmapper`

## Quick Reference
| Task | File |
|------|------|
| CLI flags | `cmd/compactmapper/main.go` |
| LAS format | `las/writer.go` |
| RGB logic | `internal/converter/converter.go` |
| GUI | `internal/gui/gui.go` |
| Sorting | `internal/sorter/sorter.go` |

## Design Decisions
1. **Go + Fyne** — **Critical driver**: non-technical users (field engineers) need GUI, not CLI. Single statically-linked binary with zero dependencies. Originally Python, rewritten for accessibility.
2. **Custom LAS writer** — no external deps, full binary control
3. **Unified binary** — single exe for GUI+CLI (GUI default for non-technical users, CLI for automation)
4. **Chunked processing** — memory-efficient large files
5. **Skip-errors mode** — handles real-world sensor glitches
6. **UTF-8 BOM handling** — Excel CSV compatibility
7. **Fixed sorting criteria** — Date/Design/MachineSettings hardcoded for IC data use case (future: make configurable)

## Documentation
Update in same commit as code:
- `README.md` - User perspective (features, CLI, usage)
- `CHANGELOG.md` - Notable changes ([Keep a Changelog](https://keepachangelog.com/))
- `ARCHITECTURE.md` - Developer perspective (structure, data flow, decisions)
- `CLAUDE.md` - Build/test commands, quick reference
