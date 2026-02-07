#!/usr/bin/env python3
"""
Sample CSV data by selecting a fixed number of rows per unique combination
of specified grouping columns.

Supports a special 'Date' key that extracts the date from the 'Time' column
(format: '2025/Oct/01 09:30:02.800').
"""

import csv
import sys
import argparse
from pathlib import Path
from datetime import datetime
from collections import defaultdict


TIME_FORMAT = '%Y/%b/%d %H:%M:%S.%f'


def parse_date_from_time(time_str: str) -> str:
    """Extract date string (YYYY-MM-DD) from Time column value."""
    try:
        dt = datetime.strptime(time_str, TIME_FORMAT)
        return dt.strftime('%Y-%m-%d')
    except ValueError:
        return time_str.split(' ')[0] if ' ' in time_str else time_str


def get_group_key(row: dict, cols: list[str]) -> tuple:
    """Build a grouping key tuple from the row based on selected columns."""
    key_parts = []
    for col in cols:
        if col == 'Date':
            time_val = row.get('Time', '')
            key_parts.append(parse_date_from_time(time_val))
        else:
            key_parts.append(row.get(col, ''))
    return tuple(key_parts)


def samplify_csv(input_file: str, output_file: str, cols: list[str], sample_count: int):
    """Sample the CSV file, keeping up to sample_count rows per group."""

    with open(input_file, 'r', encoding='utf-8-sig') as infile:
        reader = csv.DictReader(infile)
        fieldnames = reader.fieldnames

        if not fieldnames:
            print("Error: CSV file has no headers")
            sys.exit(1)

        # Validate requested columns exist (except special 'Date' key)
        for col in cols:
            if col == 'Date':
                if 'Time' not in fieldnames:
                    print("Error: 'Date' key requires 'Time' column, but it's not in the CSV")
                    sys.exit(1)
            elif col not in fieldnames:
                print(f"Error: Column '{col}' not found in CSV. Available: {', '.join(fieldnames)}")
                sys.exit(1)

        # Group rows by key
        groups = defaultdict(list)
        for row in reader:
            key = get_group_key(row, cols)
            groups[key].append(row)

    # Sample from each group
    sampled_rows = []
    total_groups = len(groups)
    short_groups = 0

    for key, rows in sorted(groups.items()):
        key_label = ', '.join(f"{c}={v}" for c, v in zip(cols, key))
        available = len(rows)

        if available < sample_count:
            short_groups += 1
            print(f"  Warning: {key_label} has only {available}/{sample_count} rows")

        sampled_rows.extend(rows[:sample_count])

    # Write output
    with open(output_file, 'w', encoding='utf-8', newline='') as outfile:
        writer = csv.DictWriter(outfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(sampled_rows)

    print(f"Sampled {len(sampled_rows)} rows from {total_groups} groups")
    if short_groups:
        print(f"  {short_groups}/{total_groups} groups had fewer than {sample_count} rows")
    print(f"Output written to: {output_file}")


def main():
    parser = argparse.ArgumentParser(
        description='Sample CSV data by selecting rows per unique column combination',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""\
Examples:
  # 10 rows per Date/DesignName/Task/Machine combination
  python samplify_data.py input.csv output.csv --cols Date,DesignName,Task,Machine

  # 5 rows per DesignName
  python samplify_data.py input.csv output.csv --cols DesignName --sample-count 5

  # 20 rows per Date/Machine (Date is parsed from Time column)
  python samplify_data.py input.csv output.csv --cols Date,Machine --sample-count 20
        """
    )

    parser.add_argument('input', help='Input CSV file path')
    parser.add_argument('output', help='Output CSV file path')
    parser.add_argument(
        '--cols',
        required=True,
        help=(
            "Comma-separated columns to group by. "
            "Special keys: 'Date' extracts the date (YYYY-MM-DD) from the 'Time' column "
            "(expected format: '2025/Oct/01 09:30:02.800'). "
            "All other names must match CSV headers exactly."
        )
    )
    parser.add_argument(
        '--sample-count',
        type=int,
        default=10,
        help='Number of rows to keep per group (default: 10)'
    )

    args = parser.parse_args()

    if not Path(args.input).exists():
        print(f"Error: Input file not found: {args.input}")
        sys.exit(1)

    cols = [c.strip() for c in args.cols.split(',') if c.strip()]
    if not cols:
        print("Error: --cols must specify at least one column")
        sys.exit(1)

    try:
        samplify_csv(args.input, args.output, cols, args.sample_count)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
