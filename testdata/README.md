# Test Data

This directory contains test fixtures for CompactMapper.

## Directory Structure

```
testdata/
├── integration/              # Integration test fixtures
│   ├── input/               # Source CSV files for testing
│   ├── expected_sorted/     # Expected sorted CSV output
│   └── expected_las/        # Expected LAS output
├── sample1.csv              # Legacy unit test fixtures
└── sample2.csv              # (to be removed)
```

## Usage

### Integration Tests
Integration tests in `test/integration_test.go` use fixtures from the `integration/` subdirectory:

- **input/** - Source CAT roller compaction CSV files
- **expected_sorted/** - Reference output from sorting operation
- **expected_las/** - Reference output from LAS conversion

### Unit Tests
Unit tests (`internal/*/sorter_test.go`, `converter_test.go`) generate test data inline using `t.TempDir()` and don't require external fixtures.

## Data Files

### Input CSVs
- `data.csv` - Main test dataset with typical CAT roller data
- `data_small.csv` - Smaller dataset for quick tests
- `data_issue.csv` - Edge cases and error scenarios
- Anonymized variants (`*_anon.csv`) - Privacy-safe versions

### Expected Outputs
Files in `expected_sorted/` and `expected_las/` represent the correct output for validation during testing.

## Maintenance

When updating test data:
1. Place new source CSVs in `integration/input/`
2. Generate expected outputs using the tool
3. Place sorted CSVs in `expected_sorted/`
4. Place LAS files in `expected_las/`
5. Update integration tests to reference new files

## Data Generation

To generate anonymized test data, use tools in `tools/data/`:
```bash
cd tools/data
python anonymize_data.py
python samplify_data.py
```
