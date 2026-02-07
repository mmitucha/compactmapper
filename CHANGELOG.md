# Changelog

All notable changes to CompactMapper will be documented in this file.

## [1.1.0] - 2025-10-03

### Added
- 🎯 **Drag & Drop Support**: Folders can now be dragged and dropped onto input/output areas
- 📋 **Full Path Display**: Complete folder paths are now shown in the GUI instead of just folder names
- 📐 **Improved Layout**: Larger window size (700x450) with better spacing
- 🎨 **Visual Feedback**: Drop areas with emoji icons and clear instructions
- 📁 **File Drop Support**: If a file is dropped, its parent directory is automatically used

### Changed
- GUI window resized from 500x300 to 700x450 for better visibility
- Label text wrapping enabled for long paths
- Enhanced user experience with clearer visual hierarchy

### Technical Details
- Implemented custom `DropArea` widget with `DraggedFiles` interface
- Added proper closure handling for drag & drop callbacks
- Full path preservation in UI labels

## [1.0.0] - 2025-10-03

### Added
- Initial release of CompactMapper
- GUI application with Fyne framework
- CLI tool for automation
- Custom LAS 1.2 writer implementation
- RGB color coding based on pass counts:
  - Red: PassCount < TargetPassCount
  - Green: PassCount = TargetPassCount
  - Blue: PassCount > TargetPassCount
- Cross-platform support (Windows, macOS, Linux)
- Batch processing of CSV files
- Real-time progress tracking
- Comprehensive test suite
- Sample test data
- Build automation with Makefile
- Documentation (README, ARCHITECTURE)

### Features
- Point Data Format 2 (XYZ + Intensity + RGB)
- LAS 1.2 specification compliance
- Automatic bounding box calculation
- Error handling with row-specific messages
- GUI and CLI variants

### Performance
- Fast CSV parsing
- Efficient binary LAS writing
- Low memory footprint
- Single executable distribution

---

## Upgrade Guide

### From 1.0.0 to 1.1.0

No breaking changes. Simply replace your executable with the new version.

**New Features Available:**
1. You can now drag folders directly onto the drop areas instead of clicking buttons
2. Full paths are displayed so you can verify the correct folders are selected
3. Larger window for better visibility

**Backwards Compatibility:**
- All existing functionality remains unchanged
- CLI tool unchanged
- LAS file format unchanged
- CSV format requirements unchanged

---

## Future Plans

### Planned Features
- [ ] LAZ (compressed LAS) support
- [ ] Custom color schemes
- [ ] Point cloud preview
- [ ] CRS/projection support
- [ ] Intensity value mapping
- [ ] Classification code options
- [ ] Progress bar with percentage
- [ ] Batch CLI mode (multiple input directories)
- [ ] Config file support

### Under Consideration
- [ ] Web-based version
- [ ] COPC output format
- [ ] E57 format support
- [ ] Point filtering options
- [ ] Statistics export
- [ ] 3D preview window
