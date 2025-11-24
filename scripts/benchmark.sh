#!/bin/bash

# LauraDB Benchmark Runner and Comparison Tool
#
# This script helps run benchmarks and compare results between commits/branches
# to detect performance regressions.

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
BENCH_DIR="benchmarks"
mkdir -p "$BENCH_DIR"

# Functions
show_help() {
    cat << EOF
LauraDB Benchmark Tool

Usage: $0 [command] [options]

Commands:
    run                 Run benchmarks and save results
    compare [old] [new] Compare two benchmark results
    baseline            Create a baseline benchmark for current code
    check               Run benchmarks and compare with baseline
    clean               Remove old benchmark results
    help                Show this help message

Examples:
    # Run benchmarks and save results
    $0 run

    # Create a baseline for the current code
    $0 baseline

    # Run benchmarks and compare with baseline
    $0 check

    # Compare two specific benchmark files
    $0 compare benchmarks/baseline.txt benchmarks/current.txt

    # Clean old results
    $0 clean

Options:
    -p, --package [pkg]  Run benchmarks for specific package only
    -t, --time [dur]     Set benchmark time (default: 3s)
    -c, --count [n]      Number of benchmark runs (default: 1)
    -v, --verbose        Verbose output

EOF
}

run_benchmarks() {
    local package="${1:-./pkg/...}"
    local benchtime="${2:-3s}"
    local count="${3:-1}"

    echo -e "${BLUE}Running benchmarks...${NC}"
    echo "Package: $package"
    echo "Bench time: $benchtime"
    echo "Count: $count"
    echo ""

    local timestamp=$(date +%Y%m%d-%H%M%S)
    local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local output="$BENCH_DIR/bench-${timestamp}-${commit}.txt"

    go test -bench=. -benchmem -benchtime="$benchtime" -count="$count" -run=^$ "$package" | tee "$output"

    echo ""
    echo -e "${GREEN}✓ Benchmark results saved to: $output${NC}"
    echo "$output"
}

create_baseline() {
    echo -e "${BLUE}Creating baseline benchmark...${NC}"

    if [ -f "$BENCH_DIR/baseline.txt" ]; then
        local backup="$BENCH_DIR/baseline-backup-$(date +%Y%m%d-%H%M%S).txt"
        mv "$BENCH_DIR/baseline.txt" "$backup"
        echo -e "${YELLOW}Previous baseline backed up to: $backup${NC}"
    fi

    go test -bench=. -benchmem -benchtime=3s -count=5 -run=^$ ./pkg/... > "$BENCH_DIR/baseline.txt" 2>&1

    echo -e "${GREEN}✓ Baseline created: $BENCH_DIR/baseline.txt${NC}"
    echo ""
    echo "Baseline summary:"
    grep "^Benchmark" "$BENCH_DIR/baseline.txt" | head -10
}

run_and_compare() {
    echo -e "${BLUE}Running benchmarks and comparing with baseline...${NC}"

    if [ ! -f "$BENCH_DIR/baseline.txt" ]; then
        echo -e "${RED}Error: No baseline found. Run '$0 baseline' first.${NC}"
        exit 1
    fi

    local current="$BENCH_DIR/current.txt"
    go test -bench=. -benchmem -benchtime=3s -count=5 -run=^$ ./pkg/... > "$current" 2>&1

    echo -e "${GREEN}✓ Current benchmarks complete${NC}"
    echo ""

    compare_benchmarks "$BENCH_DIR/baseline.txt" "$current"
}

compare_benchmarks() {
    local old="$1"
    local new="$2"

    if [ ! -f "$old" ]; then
        echo -e "${RED}Error: Old benchmark file not found: $old${NC}"
        exit 1
    fi

    if [ ! -f "$new" ]; then
        echo -e "${RED}Error: New benchmark file not found: $new${NC}"
        exit 1
    fi

    echo -e "${BLUE}Comparing benchmarks:${NC}"
    echo "  Old: $old"
    echo "  New: $new"
    echo ""

    # Check if benchstat is available
    if command -v benchstat &> /dev/null; then
        echo -e "${GREEN}Using benchstat for comparison:${NC}"
        echo ""
        benchstat "$old" "$new"
    else
        echo -e "${YELLOW}Note: Install benchstat for detailed comparison:${NC}"
        echo "  go install golang.org/x/perf/cmd/benchstat@latest"
        echo ""
        echo -e "${BLUE}Basic comparison:${NC}"
        echo ""

        # Extract benchmark names and compare
        echo "Showing side-by-side comparison of key metrics:"
        echo ""
        printf "%-50s %-20s %-20s\n" "Benchmark" "Old (ns/op)" "New (ns/op)"
        printf "%-50s %-20s %-20s\n" "$(printf '=%.0s' {1..50})" "$(printf '=%.0s' {1..20})" "$(printf '=%.0s' {1..20})"

        # Parse both files and compare (simplified version)
        grep "^Benchmark" "$old" | head -20 | while read -r line; do
            bench_name=$(echo "$line" | awk '{print $1}')
            old_nsop=$(echo "$line" | awk '{print $3}')

            new_line=$(grep "^$bench_name" "$new" | head -1)
            if [ -n "$new_line" ]; then
                new_nsop=$(echo "$new_line" | awk '{print $3}')
                printf "%-50s %-20s %-20s\n" "$bench_name" "$old_nsop" "$new_nsop"
            fi
        done

        echo ""
        echo -e "${BLUE}Old results summary:${NC}"
        grep "^Benchmark" "$old" | head -10
        echo ""
        echo -e "${BLUE}New results summary:${NC}"
        grep "^Benchmark" "$new" | head -10
    fi
}

clean_old_results() {
    echo -e "${BLUE}Cleaning old benchmark results...${NC}"

    # Keep baseline and current, remove others older than 30 days
    find "$BENCH_DIR" -name "bench-*.txt" -type f -mtime +30 -exec rm {} \;

    local count=$(find "$BENCH_DIR" -name "bench-*.txt" -type f | wc -l)
    echo -e "${GREEN}✓ Cleanup complete. Remaining benchmark files: $count${NC}"
}

# Main script
case "${1:-help}" in
    run)
        run_benchmarks "${2:-./pkg/...}" "${3:-3s}" "${4:-1}"
        ;;
    baseline)
        create_baseline
        ;;
    check)
        run_and_compare
        ;;
    compare)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo -e "${RED}Error: compare requires two file arguments${NC}"
            echo "Usage: $0 compare <old-file> <new-file>"
            exit 1
        fi
        compare_benchmarks "$2" "$3"
        ;;
    clean)
        clean_old_results
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
