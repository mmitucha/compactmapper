#!/usr/bin/env python3
"""
Visualize CAT compactor telemetry data from CSV files ONLY.

This script analyzes compaction quality by comparing PassCount vs TargetPassCount.
It generates heatmaps showing under-compacted (red), optimal (green), and
over-compacted (blue) areas in UTM coordinates.

IMPORTANT: Only CSV files are supported. Use LAS files directly with GIS software
or convert them to CSV first.
"""

import argparse
import os
import pandas as pd
import matplotlib.pyplot as plt
from matplotlib.colors import Normalize, ListedColormap
import numpy as np


def load_data(file_path):
    """Load compaction data from CSV file only.

    Args:
        file_path: Path to CSV file with compaction data

    Returns:
        DataFrame with compaction data and quality metrics

    Raises:
        ValueError: If file is not a CSV file
        FileNotFoundError: If file doesn't exist
    """
    # Check file exists
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"File not found: {file_path}")

    # Check file extension
    _, ext = os.path.splitext(file_path.lower())
    if ext != '.csv':
        raise ValueError(
            f"ERROR: Only CSV files are supported. Got '{ext}' file.\n"
            f"This script expects CSV format with columns: "
            f"CellE_m, CellN_m, Elevation_m, PassCount, TargPassCount.\n"
            f"If you have a LAS file, please convert it to CSV first or use GIS software."
        )

    # Read CSV
    df = pd.read_csv(file_path, na_values=['?', ''])

    # Convert required columns to proper types
    required_cols = ['CellE_m', 'CellN_m', 'Elevation_m', 'PassCount', 'TargPassCount']
    for col in required_cols:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors='coerce')

    # Convert optional columns to proper types
    optional_cols = ['LastCMV', 'TargCMV', 'LastMDP', 'TargMDP', 'LastRMV',
                    'LastFreq', 'LastAmp', 'LastTemp', 'LastEVIB1', 'TargEVIB1',
                    'LastEVIB2', 'TargEVIB2', 'TargThickness']

    for col in optional_cols:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors='coerce')

    # Calculate compaction quality (delta from target)
    if 'PassCount' in df.columns and 'TargPassCount' in df.columns:
        df['CompactionDelta'] = df['PassCount'] - df['TargPassCount']
        df['CompactionQuality'] = df['CompactionDelta'].apply(classify_compaction)

    return df


def classify_compaction(delta):
    """Classify compaction quality based on delta from target.
    Returns: -1 (under), 0 (optimal), 1 (over)
    """
    if pd.isna(delta):
        return 0
    elif delta < 0:
        return -1  # Under-compacted (red)
    elif delta == 0:
        return 0   # Optimal (green)
    else:
        return 1   # Over-compacted (blue)


def plot_compaction_quality(df, output_file=None):
    """Visualize compaction quality heatmap: red=under, green=optimal, blue=over target.
    UTM coordinate heatmap with filled color zones.
    """
    from scipy.interpolate import griddata

    fig, ax = plt.subplots(figsize=(16, 12))

    # Get coordinate ranges
    x = df['CellE_m'].values
    y = df['CellN_m'].values
    z = df['CompactionDelta'].values  # Use delta for gradient

    # Create grid for interpolation
    # Use more grid points for smoother heatmap
    x_range = x.max() - x.min()
    y_range = y.max() - y.min()

    # Handle case where data is in a line (x_range or y_range is 0)
    if x_range < 0.1:
        x_range = 5.0  # Add padding
    if y_range < 0.1:
        y_range = 5.0

    grid_x, grid_y = np.mgrid[
        x.min()-x_range*0.1:x.max()+x_range*0.1:200j,
        y.min()-y_range*0.1:y.max()+y_range*0.1:200j
    ]

    # Interpolate data onto grid
    grid_z = griddata((x, y), z, (grid_x, grid_y), method='nearest')

    # Check if all values are the same
    if grid_z.max() - grid_z.min() < 0.01:
        # Uniform data - use imshow instead of contourf
        # Choose color based on the value
        if z[0] < 0:
            fill_color = '#FF3333'  # Red for under
        elif z[0] == 0:
            fill_color = '#33FF33'  # Green for optimal
        else:
            fill_color = '#3333FF'  # Blue for over

        # Fill the entire area with single color
        ax.imshow([[0]], extent=[x.min()-x_range*0.1, x.max()+x_range*0.1,
                                 y.min()-y_range*0.1, y.max()+y_range*0.1],
                  cmap=ListedColormap([fill_color]), alpha=0.3, aspect='auto')
        contourf = None
    else:
        # Varying data - use contourf
        colors_map = ['#FF3333', '#FFFF33', '#33FF33', '#3399FF', '#3333FF']
        n_bins = 5
        cmap = ListedColormap(colors_map)
        levels = np.linspace(grid_z.min(), grid_z.max(), n_bins+1)
        contourf = ax.contourf(grid_x, grid_y, grid_z, levels=levels, cmap=cmap, alpha=0.7)

    # Overlay actual data points
    scatter = ax.scatter(x, y, c=df['CompactionQuality'],
                        cmap=ListedColormap(['#AA0000', '#00AA00', '#0000AA']),
                        vmin=-1, vmax=1,
                        s=200, alpha=1.0,
                        edgecolors='black', linewidths=2,
                        marker='o', zorder=5)

    # Colorbar with clear labels (only if we have varying data)
    if contourf is not None:
        cbar = plt.colorbar(contourf, ax=ax)
        cbar.set_label('Pass Count Delta from Target', rotation=270, labelpad=20, fontsize=11)

    # Count statistics
    under = (df['CompactionQuality'] == -1).sum()
    optimal = (df['CompactionQuality'] == 0).sum()
    over = (df['CompactionQuality'] == 1).sum()

    # Add grid
    ax.grid(True, alpha=0.3, linestyle='--', linewidth=0.5, color='gray', zorder=0)

    # Labels
    ax.set_xlabel('UTM Easting (m)', fontsize=12, fontweight='bold')
    ax.set_ylabel('UTM Northing (m)', fontsize=12, fontweight='bold')

    # Status message
    status = []
    if under > 0:
        status.append(f'⚠️  {under} UNDER-COMPACTED')
    if over > 0:
        status.append(f'⚠️  {over} OVER-COMPACTED')
    if optimal == len(df):
        status.append('✓ ALL OPTIMAL')

    status_msg = ' | '.join(status) if status else 'Analysis complete'
    ax.set_title(f'Compaction Quality Heatmap (UTM Coordinates)\n{status_msg}',
                fontsize=14, fontweight='bold', pad=20)

    # Legend
    from matplotlib.patches import Patch
    legend_elements = [
        Patch(facecolor='#FF3333', edgecolor='black', label=f'Under Target ({under} pts)'),
        Patch(facecolor='#33FF33', edgecolor='black', label=f'At Target ({optimal} pts)'),
        Patch(facecolor='#3333FF', edgecolor='black', label=f'Over Target ({over} pts)')
    ]
    ax.legend(handles=legend_elements, loc='upper right', fontsize=11,
             framealpha=0.95, title='Compaction Status')

    # Statistics box
    stats_text = f'Total Points: {len(df)}\n'
    if 'TargPassCount' in df.columns and df['TargPassCount'].notna().any():
        target = int(df['TargPassCount'].iloc[0])
        stats_text += f'Target: {target} passes\n'
    stats_text += f'Actual: {int(df["PassCount"].min())}-{int(df["PassCount"].max())}'

    ax.text(0.02, 0.98, stats_text,
           transform=ax.transAxes, verticalalignment='top',
           bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.9),
           fontsize=10, family='monospace')

    # Equal aspect for UTM
    ax.set_aspect('equal')

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def plot_pass_count(df, output_file=None):
    """Visualize pass count distribution."""
    fig, ax = plt.subplots(figsize=(12, 8))

    scatter = ax.scatter(df['CellE_m'], df['CellN_m'],
                        c=df['PassCount'],
                        cmap='viridis',
                        s=50,
                        alpha=0.7,
                        edgecolors='none')

    plt.colorbar(scatter, ax=ax, label='Pass Count')
    ax.set_xlabel('Easting (m)')
    ax.set_ylabel('Northing (m)')
    ax.set_title('Compaction Pass Count Distribution')
    ax.grid(True, alpha=0.3)
    ax.set_aspect('equal')

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def plot_elevation(df, output_file=None):
    """Visualize elevation."""
    fig, ax = plt.subplots(figsize=(12, 8))

    scatter = ax.scatter(df['CellE_m'], df['CellN_m'],
                        c=df['Elevation_m'],
                        cmap='terrain',
                        s=50,
                        alpha=0.7,
                        edgecolors='none')

    plt.colorbar(scatter, ax=ax, label='Elevation (m)')
    ax.set_xlabel('Easting (m)')
    ax.set_ylabel('Northing (m)')
    ax.set_title('Elevation Map')
    ax.grid(True, alpha=0.3)
    ax.set_aspect('equal')

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def plot_cmv(df, output_file=None):
    """Visualize Compaction Meter Value (CMV)."""
    fig, ax = plt.subplots(figsize=(12, 8))

    # Filter out NaN values
    valid_data = df.dropna(subset=['LastCMV'])

    scatter = ax.scatter(valid_data['CellE_m'], valid_data['CellN_m'],
                        c=valid_data['LastCMV'],
                        cmap='RdYlGn',
                        s=50,
                        alpha=0.7,
                        edgecolors='none')

    plt.colorbar(scatter, ax=ax, label='CMV')
    ax.set_xlabel('Easting (m)')
    ax.set_ylabel('Northing (m)')
    ax.set_title('Compaction Meter Value (CMV)')
    ax.grid(True, alpha=0.3)
    ax.set_aspect('equal')

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def plot_mdp(df, output_file=None):
    """Visualize Machine Drive Power (MDP)."""
    fig, ax = plt.subplots(figsize=(12, 8))

    # Filter out NaN values
    valid_data = df.dropna(subset=['LastMDP'])

    scatter = ax.scatter(valid_data['CellE_m'], valid_data['CellN_m'],
                        c=valid_data['LastMDP'],
                        cmap='plasma',
                        s=50,
                        alpha=0.7,
                        edgecolors='none')

    plt.colorbar(scatter, ax=ax, label='MDP')
    ax.set_xlabel('Easting (m)')
    ax.set_ylabel('Northing (m)')
    ax.set_title('Machine Drive Power (MDP)')
    ax.grid(True, alpha=0.3)
    ax.set_aspect('equal')

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def plot_overview(df, output_file=None):
    """Create a 2x2 overview plot focusing on compaction quality."""
    fig, axes = plt.subplots(2, 2, figsize=(16, 14))

    # Compaction Quality (most important)
    colors = ['#FF4444', '#44FF44', '#4444FF']
    cmap = ListedColormap(colors)
    scatter1 = axes[0, 0].scatter(df['CellE_m'], df['CellN_m'],
                                  c=df['CompactionQuality'],
                                  cmap=cmap,
                                  vmin=-1, vmax=1,
                                  s=30,
                                  alpha=0.8,
                                  edgecolors='none')
    cbar1 = plt.colorbar(scatter1, ax=axes[0, 0], ticks=[-1, 0, 1])
    cbar1.ax.set_yticklabels(['Under', 'Optimal', 'Over'])
    axes[0, 0].set_xlabel('Easting (m)')
    axes[0, 0].set_ylabel('Northing (m)')
    axes[0, 0].set_title('Compaction Quality (Red=Under, Green=Optimal, Blue=Over)')
    axes[0, 0].grid(True, alpha=0.3)
    axes[0, 0].set_aspect('equal')

    # Pass Count Distribution
    scatter2 = axes[0, 1].scatter(df['CellE_m'], df['CellN_m'],
                                  c=df['PassCount'],
                                  cmap='viridis',
                                  s=30,
                                  alpha=0.7,
                                  edgecolors='none')
    plt.colorbar(scatter2, ax=axes[0, 1], label='Pass Count')
    axes[0, 1].set_xlabel('Easting (m)')
    axes[0, 1].set_ylabel('Northing (m)')
    axes[0, 1].set_title('Pass Count Distribution')
    axes[0, 1].grid(True, alpha=0.3)
    axes[0, 1].set_aspect('equal')

    # Elevation
    scatter3 = axes[1, 0].scatter(df['CellE_m'], df['CellN_m'],
                                  c=df['Elevation_m'],
                                  cmap='terrain',
                                  s=30,
                                  alpha=0.7,
                                  edgecolors='none')
    plt.colorbar(scatter3, ax=axes[1, 0], label='Elevation (m)')
    axes[1, 0].set_xlabel('Easting (m)')
    axes[1, 0].set_ylabel('Northing (m)')
    axes[1, 0].set_title('Elevation Map')
    axes[1, 0].grid(True, alpha=0.3)
    axes[1, 0].set_aspect('equal')

    # Compaction Delta (how far from target)
    scatter4 = axes[1, 1].scatter(df['CellE_m'], df['CellN_m'],
                                  c=df['CompactionDelta'],
                                  cmap='RdYlGn_r',  # Reversed: red=bad, green=good
                                  s=30,
                                  alpha=0.7,
                                  edgecolors='none')
    plt.colorbar(scatter4, ax=axes[1, 1], label='Delta from Target')
    axes[1, 1].set_xlabel('Easting (m)')
    axes[1, 1].set_ylabel('Northing (m)')
    axes[1, 1].set_title('Pass Count Delta from Target')
    axes[1, 1].grid(True, alpha=0.3)
    axes[1, 1].set_aspect('equal')

    plt.tight_layout()

    if output_file:
        plt.savefig(output_file, dpi=300, bbox_inches='tight')
        print(f"Saved to {output_file}")
    else:
        plt.show()

    plt.close()


def print_statistics(df):
    """Print statistics about compaction data."""
    print("\n=== Compaction Statistics ===")
    print(f"Total points: {len(df)}")

    print(f"\nCoordinate range:")
    print(f"  Easting:  {df['CellE_m'].min():.2f} - {df['CellE_m'].max():.2f} m")
    print(f"  Northing: {df['CellN_m'].min():.2f} - {df['CellN_m'].max():.2f} m")
    print(f"  Elevation: {df['Elevation_m'].min():.2f} - {df['Elevation_m'].max():.2f} m")

    print(f"\n=== Compaction Quality ===")
    if 'TargPassCount' in df.columns and df['TargPassCount'].notna().any():
        target = df['TargPassCount'].iloc[0]
        print(f"Target pass count: {target}")

    print(f"\nPass count:")
    print(f"  Min: {df['PassCount'].min()}")
    print(f"  Max: {df['PassCount'].max()}")
    print(f"  Mean: {df['PassCount'].mean():.1f}")
    print(f"  Median: {df['PassCount'].median():.1f}")

    if 'CompactionQuality' in df.columns:
        under = (df['CompactionQuality'] == -1).sum()
        optimal = (df['CompactionQuality'] == 0).sum()
        over = (df['CompactionQuality'] == 1).sum()
        total = len(df)

        print(f"\nCompaction quality assessment:")
        print(f"  Under-compacted:  {under:5d} points ({100*under/total:5.1f}%) - NEEDS MORE PASSES")
        print(f"  Optimal:          {optimal:5d} points ({100*optimal/total:5.1f}%) - GOOD")
        print(f"  Over-compacted:   {over:5d} points ({100*over/total:5.1f}%) - EXCESSIVE")

        if under > 0:
            print(f"\n⚠️  WARNING: {under} points are under-compacted and need additional passes")
        if optimal == total:
            print(f"\n✓ EXCELLENT: All points meet target compaction")
        elif optimal / total > 0.95:
            print(f"\n✓ GOOD: {100*optimal/total:.1f}% of points meet target compaction")

    # Additional sensor data (if available)
    if 'LastCMV' in df.columns and df['LastCMV'].notna().any():
        print("\n=== Additional Sensor Data ===")
        print(f"CMV (Compaction Meter Value):")
        print(f"  Valid points: {df['LastCMV'].notna().sum()} / {len(df)}")
        print(f"  Mean: {df['LastCMV'].mean():.1f}")

    if 'LastMDP' in df.columns and df['LastMDP'].notna().any():
        print(f"MDP (Machine Drive Power):")
        print(f"  Valid points: {df['LastMDP'].notna().sum()} / {len(df)}")
        print(f"  Mean: {df['LastMDP'].mean():.1f}")
    print()


def main():
    parser = argparse.ArgumentParser(
        description='Visualize CAT compactor telemetry data from CSV files - analyze compaction quality',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --file data.csv --stats
  %(prog)s --file data.csv --type quality --output quality_map.png
  %(prog)s --file data.csv --type overview

Note:
  This script ONLY accepts CSV files. LAS files are not supported.
  Required CSV columns: CellE_m, CellN_m, Elevation_m, PassCount, TargPassCount
        """
    )

    parser.add_argument('--file', '-f',
                        required=True,
                        metavar='CSV_FILE',
                        help='Path to CSV file with compaction data (CSV format only)')

    parser.add_argument('--type', '-t',
                        choices=['overview', 'quality', 'pass_count', 'elevation', 'cmv', 'mdp'],
                        default='quality',
                        help='Type of visualization (default: quality)')

    parser.add_argument('--output', '-o',
                        help='Output file path (if not specified, shows interactive plot)')

    parser.add_argument('--stats', '-s',
                        action='store_true',
                        help='Print compaction quality statistics')

    args = parser.parse_args()

    # Load data
    print(f"Loading data from {args.file}...")
    df = load_data(args.file)
    print(f"Loaded {len(df)} data points")

    # Print statistics if requested
    if args.stats:
        print_statistics(df)

    # Create visualization
    print(f"Creating {args.type} visualization...")

    if args.type == 'overview':
        plot_overview(df, args.output)
    elif args.type == 'quality':
        plot_compaction_quality(df, args.output)
    elif args.type == 'pass_count':
        plot_pass_count(df, args.output)
    elif args.type == 'elevation':
        plot_elevation(df, args.output)
    elif args.type == 'cmv':
        plot_cmv(df, args.output)
    elif args.type == 'mdp':
        plot_mdp(df, args.output)

    if not args.output:
        print("Close the plot window to exit.")


if __name__ == '__main__':
    main()
