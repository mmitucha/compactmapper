import pandas as pd
import os
import pylas
import argparse


def main():
    parser = argparse.ArgumentParser(description='Convert CSV files to LAS format')
    parser.add_argument('--input-dir', '-i', required=True, help='Input directory containing CSV files')
    parser.add_argument('--output-dir', '-o', required=True, help='Output directory for LAS files')
    args = parser.parse_args()

    input_folder = getattr(args, 'input_dir')
    output_folder = getattr(args, 'output_dir')

    if not os.path.exists(input_folder):
        print(f"Error: Input directory '{input_folder}' does not exist")
        return

    if not os.path.exists(output_folder):
        os.makedirs(output_folder)
        print(f"Created output directory: {output_folder}")

    for filename in os.listdir(input_folder):
        if filename.endswith('.csv'):
            file_path = os.path.join(input_folder, filename)

            print(f"Loading file: {filename}")
            df = pd.read_csv(file_path, low_memory=False)

            points = []
            for _, row in df.iterrows():
                if row['PassCount'] < row['TargPassCount']:
                    color = (255, 0, 0)  # Červená
                elif row['PassCount'] == row['TargPassCount']:
                    color = (0, 255, 0)  # Zelená
                else:
                    color = (0, 0, 255)  # Modrá

                points.append((row['CellE_m'], row['CellN_m'], row['Elevation_m'], *color))

            # Vytvorenie LAS súboru so správnym formátom pre farby
            las = pylas.create(point_format_id=3)  # Verzia, ktorá podporuje RGB
            las.x, las.y, las.z = zip(*[(p[0], p[1], p[2]) for p in points])
            las.red, las.green, las.blue = zip(*[(p[3], p[4], p[5]) for p in points])
            output_file_path = os.path.join(output_folder, filename.replace('.csv', '.las'))
            las.write(output_file_path)

    print("Conversion done 🎉")

if __name__ == '__main__':
    main()
