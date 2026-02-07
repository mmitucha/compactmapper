import argparse
import pandas as pd
from datetime import datetime
import os
import re
import glob

def sanitize_filename(filename):
    return re.sub(r'[<>:"/\\|?*]', '', filename)

def process_file(file_path, output_directory):
    print(f"Processing file: {file_path}")

    # Read entire CSV file at once, keeping all values as strings to preserve precision
    df = pd.read_csv(file_path, low_memory=False, dtype=str)

    # Add Date column
    df['Date'] = df['Time'].apply(lambda x: datetime.strptime(x, '%Y/%b/%d %H:%M:%S.%f').strftime('%Y-%m-%d'))

    # Group by Date, DesignName, and LastAmp
    for (date, design, amp), group in df.groupby(['Date', 'DesignName', 'LastAmp']):
        amp_suffix = amp.replace('.', '')[:3] if pd.notnull(amp) else 'no_amp'
        filename = f'{date}design{design}amp{amp_suffix}.csv'
        sanitized_filename = sanitize_filename(filename)
        full_path = os.path.join(output_directory, sanitized_filename)

        # Drop Date column before writing
        output_group = group.drop(columns=['Date'])
        # Write entire group to file at once
        output_group.to_csv(full_path, index=False, header=True)

    print(f"Completed: {file_path}")

def main():
    parser = argparse.ArgumentParser(description='Process CSV files and split by Date, DesignName, and LastAmp')
    parser.add_argument('--files', required=True, help='Wildcard path to CSV files (e.g., "data/*.csv" or "data/file*.csv")')
    parser.add_argument('--output', required=True, help='Output directory for processed files')

    args = parser.parse_args()

    # Expand wildcard to get list of files
    file_list = glob.glob(args.files)

    if not file_list:
        print(f"No files found matching pattern: {args.files}")
        return

    # Create output directory if it doesn't exist
    os.makedirs(args.output, exist_ok=True)

    print(f"Found {len(file_list)} file(s) to process")

    # Process each file
    for file_path in file_list:
        process_file(file_path, args.output)

    print(f"\nAll files processed successfully! Output saved to: {args.output}")

if __name__ == "__main__":
    main()
