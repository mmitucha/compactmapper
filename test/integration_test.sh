#!/bin/bash

# CompactMapper Integration Test Script
# This script tests the full pipeline: CSV -> Sort -> LAS conversion
# and compares outputs with expected results in sample/ directory

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TESTDATA_DIR="$PROJECT_ROOT/testdata/e2e"
INPUT_CSV="$TESTDATA_DIR/input/data.csv"
EXPECTED_SORTED="$TESTDATA_DIR/expected_sorted"
EXPECTED_LAS="$TESTDATA_DIR/expected_las"
TEST_OUTPUT_DIR="$PROJECT_ROOT/tmp/integration"
BINARY="$PROJECT_ROOT/build/compactmapper"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Test helper
assert_file_exists() {
    local file="$1"
    local description="$2"

    if [ -f "$file" ]; then
        print_success "$description: File exists"
        ((TESTS_PASSED++))
        return 0
    else
        print_error "$description: File not found: $file"
        ((TESTS_FAILED++))
        return 1
    fi
}

assert_directory_not_empty() {
    local dir="$1"
    local pattern="$2"
    local description="$3"

    local file_count=$(find "$dir" -name "$pattern" -type f 2>/dev/null | wc -l)

    if [ "$file_count" -gt 0 ]; then
        print_success "$description: Found $file_count files"
        ((TESTS_PASSED++))
        return 0
    else
        print_error "$description: No files found matching $pattern"
        ((TESTS_FAILED++))
        return 1
    fi
}

compare_csv_files() {
    local actual_dir="$1"
    local expected_dir="$2"
    local description="$3"

    print_status "$description"

    # Count files in both directories
    local actual_count=$(find "$actual_dir" -name "*.csv" -type f 2>/dev/null | wc -l)
    local expected_count=$(find "$expected_dir" -name "*.csv" -type f 2>/dev/null | wc -l)

    print_status "  Actual files: $actual_count"
    print_status "  Expected files: $expected_count"

    # Check if we have files
    if [ "$actual_count" -eq 0 ]; then
        print_error "  No CSV files generated"
        ((TESTS_FAILED++))
        return 1
    fi

    # Check that each expected file exists in actual output and matches
    local matched=0
    local missing=0
    local mismatched=0

    for expected_file in "$expected_dir"/*.csv; do
        if [ ! -f "$expected_file" ]; then
            continue
        fi

        local filename=$(basename "$expected_file")
        local actual_file="$actual_dir/$filename"

        if [ -f "$actual_file" ]; then
            # Compare file line counts
            local actual_lines=$(wc -l < "$actual_file" | tr -d ' ')
            local expected_lines=$(wc -l < "$expected_file" | tr -d ' ')

            if [ "$actual_lines" -eq "$expected_lines" ]; then
                ((matched++))
                print_status "  $filename: $actual_lines lines ✓"
            else
                ((mismatched++))
                print_error "  $filename: $actual_lines lines (expected: $expected_lines) ✗"
            fi
        else
            ((missing++))
            print_warning "  Missing file: $filename"
        fi
    done

    if [ "$mismatched" -gt 0 ]; then
        print_error "$description: $mismatched files mismatched"
        ((TESTS_FAILED++))
        return 1
    elif [ "$matched" -gt 0 ]; then
        print_success "$description: $matched files matched"
        ((TESTS_PASSED++))
        return 0
    else
        print_error "$description: No matching files found"
        ((TESTS_FAILED++))
        return 1
    fi
}

compare_las_files() {
    local actual_dir="$1"
    local expected_dir="$2"
    local description="$3"

    print_status "$description"

    # Count files in both directories
    local actual_count=$(find "$actual_dir" -name "*.las" -type f 2>/dev/null | wc -l)
    local expected_count=$(find "$expected_dir" -name "*.las" -type f 2>/dev/null | wc -l)

    print_status "  Actual LAS files: $actual_count"
    print_status "  Expected LAS files: $expected_count"

    # Check if we have files
    if [ "$actual_count" -eq 0 ]; then
        print_error "  No LAS files generated"
        ((TESTS_FAILED++))
        return 1
    fi

    # Check that each expected file exists in actual output and matches
    local matched=0
    local missing=0
    local mismatched=0

    for expected_file in "$expected_dir"/*.las; do
        if [ ! -f "$expected_file" ]; then
            continue
        fi

        local filename=$(basename "$expected_file")
        local actual_file="$actual_dir/$filename"

        if [ -f "$actual_file" ]; then
            # Compare file sizes (must match exactly for binary files)
            local actual_size=$(stat -f%z "$actual_file" 2>/dev/null || stat -c%s "$actual_file" 2>/dev/null)
            local expected_size=$(stat -f%z "$expected_file" 2>/dev/null || stat -c%s "$expected_file" 2>/dev/null)

            if [ "$actual_size" -eq "$expected_size" ]; then
                ((matched++))
                print_status "  $filename: $actual_size bytes ✓"
            else
                ((mismatched++))
                print_error "  $filename: $actual_size bytes (expected: $expected_size bytes) ✗"
            fi
        else
            ((missing++))
            print_warning "  Missing LAS file: $filename"
        fi
    done

    if [ "$mismatched" -gt 0 ]; then
        print_error "$description: $mismatched LAS files mismatched"
        ((TESTS_FAILED++))
        return 1
    elif [ "$matched" -gt 0 ]; then
        print_success "$description: $matched LAS files matched"
        ((TESTS_PASSED++))
        return 0
    else
        print_error "$description: No matching LAS files found"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main test execution
main() {
    echo "================================================"
    echo "CompactMapper Integration Test"
    echo "================================================"
    echo ""

    # Check prerequisites
    print_status "Checking prerequisites..."

    if [ ! -f "$INPUT_CSV" ]; then
        print_error "Input CSV not found: $INPUT_CSV"
        exit 1
    fi

    if [ ! -d "$EXPECTED_SORTED" ]; then
        print_error "Expected sorted directory not found: $EXPECTED_SORTED"
        exit 1
    fi

    if [ ! -d "$EXPECTED_LAS" ]; then
        print_error "Expected LAS directory not found: $EXPECTED_LAS"
        exit 1
    fi

    # Build the binary if it doesn't exist
    if [ ! -f "$BINARY" ]; then
        print_status "Building compactmapper binary..."
        cd "$PROJECT_ROOT"
        make build || {
            print_error "Failed to build binary"
            exit 1
        }
    fi

    assert_file_exists "$BINARY" "Binary exists"

    # Clean up previous test output
    if [ -d "$TEST_OUTPUT_DIR" ]; then
        print_status "Cleaning up previous test output..."
        rm -rf "$TEST_OUTPUT_DIR"
    fi

    mkdir -p "$TEST_OUTPUT_DIR"

    echo ""
    echo "================================================"
    echo "Test 1: Sort CSV Only"
    echo "================================================"

    local sorted_output="$TEST_OUTPUT_DIR/sorted"

    print_status "Running: $BINARY --input $INPUT_CSV --output $sorted_output --sort-only"
    "$BINARY" --input "$INPUT_CSV" --output "$sorted_output" --sort-only

    echo ""
    assert_directory_not_empty "$sorted_output" "*.csv" "Sorted CSV files generated"
    compare_csv_files "$sorted_output" "$EXPECTED_SORTED" "Compare sorted output with expected"

    echo ""
    echo "================================================"
    echo "Test 2: Convert Sorted CSV to LAS"
    echo "================================================"

    local las_output="$TEST_OUTPUT_DIR/las"

    print_status "Running: $BINARY --input $sorted_output --output $las_output --convert-only"
    "$BINARY" --input "$sorted_output" --output "$las_output" --convert-only

    echo ""
    assert_directory_not_empty "$las_output" "*.las" "LAS files generated"
    compare_las_files "$las_output" "$EXPECTED_LAS" "Compare LAS output with expected"

    echo ""
    echo "================================================"
    echo "Test 3: Full Pipeline (Sort + Convert)"
    echo "================================================"

    local full_output="$TEST_OUTPUT_DIR/full_pipeline"

    print_status "Running: $BINARY --input $INPUT_CSV --output $full_output"
    "$BINARY" --input "$INPUT_CSV" --output "$full_output"

    echo ""
    assert_directory_not_empty "$full_output/csv" "*.csv" "Full pipeline: Sorted CSVs"
    assert_directory_not_empty "$full_output/las" "*.las" "Full pipeline: LAS files"
    compare_csv_files "$full_output/csv" "$EXPECTED_SORTED" "Full pipeline: Compare sorted output"
    compare_las_files "$full_output/las" "$EXPECTED_LAS" "Full pipeline: Compare LAS output"

    # Summary
    echo ""
    echo "================================================"
    echo "Test Summary"
    echo "================================================"

    local total=$((TESTS_PASSED + TESTS_FAILED))

    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED / $total"

    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}Failed:${NC} $TESTS_FAILED / $total"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

# Run main
main
