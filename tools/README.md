# Development Tools

Utilities and scripts for CompactMapper development.

## Directory Structure

```
tools/
└── data/                    # Data processing utilities
    ├── anonymize_data.py   # Anonymize sensitive data
    ├── samplify_data.py    # Generate sample datasets
    ├── requirements.txt    # Python dependencies
    └── README.md           # Data tools documentation
```

## Usage

### Data Tools
Tools for working with test data, anonymization, and sample generation.

See [tools/data/README.md](data/README.md) for detailed usage.

## Contributing

When adding new tools:
1. Create appropriate subdirectory (e.g., `tools/scripts/`, `tools/generators/`)
2. Include a README.md explaining usage
3. Document dependencies in requirements.txt or go.mod as appropriate
4. Update this README with tool overview
