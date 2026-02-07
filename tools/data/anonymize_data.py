#!/usr/bin/env python3
"""
Anonymize CSV data by:
1. Replacing sensitive text fields with fake data using Faker
2. Shifting coordinates in specified direction and distance
"""

import csv
import sys
import argparse
from pathlib import Path
from faker import Faker

# Initialize Faker
fake = Faker()

# Fields to anonymize
TEXT_FIELDS_TO_ANONYMIZE = {
    'DesignName': lambda: fake.catch_phrase().replace(' ', '-'),
    'MeasuredData': lambda: f"Dataset-{fake.word()}-{fake.random_int(1, 9999):04d}",
    'Machine': lambda: f"Machine-{fake.color_name()}-{fake.random_int(1, 99):02d}"
}


def calculate_coordinate_shift(north_south_m: float, west_east_m: float):
    """
    Calculate the shift in meters for North/East coordinates.

    Args:
        north_south_m: Distance to shift north (positive) or south (negative) in meters
        west_east_m: Distance to shift east (positive) or west (negative) in meters

    Returns:
        tuple: (north_shift_m, east_shift_m)
    """
    return (north_south_m, west_east_m)

# TODO: Refactor this function to reduce its Cognitive Complexity from 30 to the 15 allowed. [+16 locations]
def anonymize_csv(input_file: str, output_file: str, north_south_m: float = 0, west_east_m: float = 0):
    """Anonymize the CSV file."""

    # Calculate coordinate shifts
    north_shift, east_shift = calculate_coordinate_shift(north_south_m, west_east_m)

    # Store mapping for consistency within the file
    value_mapping = {}

    with open(input_file, 'r', encoding='utf-8-sig') as infile:
        # Read CSV
        reader = csv.DictReader(infile)
        fieldnames = reader.fieldnames

        if not fieldnames:
            print("Error: CSV file has no headers")
            sys.exit(1)

        rows = []
        for idx, row in enumerate(reader):
            # Anonymize text fields with consistent mapping
            for field, generator in TEXT_FIELDS_TO_ANONYMIZE.items():
                if field in row and row[field]:
                    original_value = row[field]
                    if original_value not in value_mapping:
                        value_mapping[original_value] = generator()
                    row[field] = value_mapping[original_value]

            # Shift North coordinate
            if 'CellN_m' in row and row['CellN_m']:
                try:
                    original_north = float(row['CellN_m'])
                    row['CellN_m'] = f"{original_north + north_shift:.3f}"
                except ValueError:
                    pass  # Keep original if not a valid number

            # Shift East coordinate
            if 'CellE_m' in row and row['CellE_m']:
                try:
                    original_east = float(row['CellE_m'])
                    row['CellE_m'] = f"{original_east + east_shift:.3f}"
                except ValueError:
                    pass  # Keep original if not a valid number

            rows.append(row)

    # Write anonymized data
    with open(output_file, 'w', encoding='utf-8', newline='') as outfile:
        writer = csv.DictWriter(outfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)

    print(f"✓ Anonymized {len(rows)} rows")
    print("✓ Coordinate shifts applied:")
    print(f"  - North/South: {north_shift:+.0f}m ({'north' if north_shift > 0 else 'south' if north_shift < 0 else 'no shift'})")
    print(f"  - West/East: {east_shift:+.0f}m ({'east' if east_shift > 0 else 'west' if east_shift < 0 else 'no shift'})")
    print(f"✓ Output written to: {output_file}")


def main():
    parser = argparse.ArgumentParser(
        description='Anonymize CSV data with fake values and coordinate shifting',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Default: shift 1km south
  python anonymize_data.py sample/0_src/data_small.csv output.csv

  # Shift 1500km south (negative = south)
  python anonymize_data.py sample/0_src/data_small.csv output.csv --north -1500000

  # Shift 500km north (positive = north)
  python anonymize_data.py sample/0_src/data_small.csv output.csv --north 500000

  # Shift 200km east (positive = east)
  python anonymize_data.py sample/0_src/data_small.csv output.csv --west 200000

  # Shift 300km west (negative = west)
  python anonymize_data.py sample/0_src/data_small.csv output.csv --west -300000

  # Shift both: 1500km south AND 300km west
  python anonymize_data.py sample/0_src/data_small.csv output.csv --north -1500000 --west -300000
        """
    )

    parser.add_argument('input', help='Input CSV file path')
    parser.add_argument('output', help='Output CSV file path')
    parser.add_argument(
        '--north',
        type=float,
        default=1000,
        help='Shift north/south in meters (positive=north, negative=south, default: 1000 = 1km north)'
    )
    parser.add_argument(
        '--west',
        type=float,
        default=1000,
        help='Shift east/west in meters (positive=east, negative=west, default: 1000 = 1km east)'
    )

    args = parser.parse_args()

    if not Path(args.input).exists():
        print(f"Error: Input file not found: {args.input}")
        sys.exit(1)

    # Use the values directly
    north_south_m = args.north
    west_east_m = args.west

    try:
        anonymize_csv(args.input, args.output, north_south_m, west_east_m)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
