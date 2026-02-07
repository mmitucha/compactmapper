# Data Processing Utilities

Python utilities for generating and processing test data for CompactMapper.

## Prerequisites

```bash
# Install Python dependencies
pip install -r requirements.txt
```

## Tools

### anonymize_data.py
Anonymizes sensitive information in CAT roller CSV files for safe sharing and testing.

**Usage:**
```bash
python anonymize_data.py <input.csv> <output.csv>
```

**What it anonymizes:**
- Machine identifiers
- Operator information (if present)
- Project-specific names
- GPS coordinates (optional shift)
- Preserves data structure and relationships

**Example:**
```bash
python anonymize_data.py ../../testdata/integration/input/data.csv ../../testdata/integration/input/data_anon.csv
```

### samplify_data.py
Generates sample datasets by extracting subsets of larger CSV files for testing.

**Usage:**
```bash
python samplify_data.py <input.csv> <output.csv> [--rows N]
```

**Options:**
- `--rows N` - Number of rows to extract (default: 100)
- `--random` - Random sampling instead of first N rows
- `--preserve-groups` - Keep complete groups by Date/DesignName/LastAmp

**Example:**
```bash
# Extract first 100 rows
python samplify_data.py ../../testdata/integration/input/data.csv ../../testdata/integration/input/data_small.csv --rows 100

# Random sample with preserved groups
python samplify_data.py ../../testdata/integration/input/data.csv ../../testdata/integration/input/data_sample.csv --rows 50 --random --preserve-groups
```

## Dependencies

- **faker** (24.0.0) - Generates realistic anonymized data

## Workflow

Typical workflow for creating test data:

```bash
# 1. Start with real data (keep privately)
# (not in repo)

# 2. Anonymize for safe inclusion in repo
python anonymize_data.py real_data.csv ../../testdata/integration/input/data.csv

# 3. Create smaller samples for quick tests
python samplify_data.py ../../testdata/integration/input/data.csv ../../testdata/integration/input/data_small.csv --rows 100

# 4. Run the tool to generate expected outputs
cd ../..
make build
./compactmapper --input testdata/integration/input/data.csv --output testdata/integration/expected_sorted --sort-only
./compactmapper --input testdata/integration/expected_sorted --output testdata/integration/expected_las --convert-only

# 5. Run tests to verify
make test-integration
```

## Notes

- Never commit real, non-anonymized data to the repository
- Test data should be representative but not production data
- Maintain privacy and confidentiality of customer data
