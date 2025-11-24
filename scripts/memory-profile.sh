#!/bin/bash

# Memory Profiling Script for LauraDB
# This script helps detect memory leaks and analyze memory usage patterns

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROFILE_DIR="$PROJECT_ROOT/profiles"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create profiles directory if it doesn't exist
mkdir -p "$PROFILE_DIR"

print_header() {
    echo -e "${BLUE}===================================================${NC}"
    echo -e "${BLUE}  LauraDB Memory Profiling Tool${NC}"
    echo -e "${BLUE}===================================================${NC}"
    echo ""
}

print_usage() {
    cat << EOF
Usage: $0 [command] [options]

Commands:
    test                Run memory leak detection tests
    profile <package>   Generate memory profile for package
    heap <package>      Generate heap profile for package
    analyze <profile>   Analyze a memory profile with pprof
    compare <p1> <p2>   Compare two profiles
    benchmark           Run benchmarks with memory profiling
    check               Run all memory checks (tests + benchmarks)
    clean               Clean up profile files
    help                Show this help message

Examples:
    $0 test                           # Run leak detection tests
    $0 profile ./pkg/database         # Profile database package
    $0 heap ./pkg/storage             # Generate heap profile
    $0 analyze profiles/mem.prof      # Analyze profile
    $0 compare p1.prof p2.prof        # Compare two profiles
    $0 benchmark                      # Run benchmarks with profiling
    $0 check                          # Run all memory checks

EOF
}

run_leak_tests() {
    echo -e "${GREEN}Running memory leak detection tests...${NC}"
    echo ""

    cd "$PROJECT_ROOT"

    # Run memory leak tests
    if go test ./pkg/metrics -run TestMemoryLeak -v -timeout 60s; then
        echo ""
        echo -e "${GREEN}✓ All memory leak tests passed${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}✗ Memory leak tests failed${NC}"
        return 1
    fi
}

run_profile() {
    local package=$1
    if [ -z "$package" ]; then
        echo -e "${RED}Error: Package path required${NC}"
        echo "Usage: $0 profile <package>"
        exit 1
    fi

    echo -e "${GREEN}Generating memory profile for $package...${NC}"
    echo ""

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local profile_file="$PROFILE_DIR/mem_${timestamp}.prof"

    cd "$PROJECT_ROOT"

    # Run tests with memory profiling
    if go test "$package" -memprofile="$profile_file" -run=. -bench=. -benchmem; then
        echo ""
        echo -e "${GREEN}✓ Memory profile saved to: $profile_file${NC}"
        echo ""
        echo "To analyze the profile, run:"
        echo -e "${YELLOW}  go tool pprof $profile_file${NC}"
        echo -e "${YELLOW}  go tool pprof -http=:8080 $profile_file${NC}"
    else
        echo -e "${RED}✗ Profiling failed${NC}"
        return 1
    fi
}

run_heap_profile() {
    local package=$1
    if [ -z "$package" ]; then
        echo -e "${RED}Error: Package path required${NC}"
        echo "Usage: $0 heap <package>"
        exit 1
    fi

    echo -e "${GREEN}Generating heap profile for $package...${NC}"
    echo ""

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local profile_file="$PROFILE_DIR/heap_${timestamp}.prof"

    cd "$PROJECT_ROOT"

    # Run tests with heap profiling
    if go test "$package" -memprofile="$profile_file" -memprofilerate=1; then
        echo ""
        echo -e "${GREEN}✓ Heap profile saved to: $profile_file${NC}"
        echo ""
        echo "To analyze the heap, run:"
        echo -e "${YELLOW}  go tool pprof -alloc_space $profile_file${NC}"
        echo -e "${YELLOW}  go tool pprof -inuse_space $profile_file${NC}"
    else
        echo -e "${RED}✗ Heap profiling failed${NC}"
        return 1
    fi
}

analyze_profile() {
    local profile=$1
    if [ -z "$profile" ]; then
        echo -e "${RED}Error: Profile file required${NC}"
        echo "Usage: $0 analyze <profile>"
        exit 1
    fi

    if [ ! -f "$profile" ]; then
        echo -e "${RED}Error: Profile file not found: $profile${NC}"
        exit 1
    fi

    echo -e "${GREEN}Analyzing profile: $profile${NC}"
    echo ""

    # Show top memory allocations
    echo -e "${BLUE}Top memory allocations:${NC}"
    go tool pprof -top "$profile" 2>/dev/null | head -20
    echo ""

    echo -e "${BLUE}To open interactive analysis:${NC}"
    echo -e "${YELLOW}  go tool pprof -http=:8080 $profile${NC}"
}

compare_profiles() {
    local p1=$1
    local p2=$2

    if [ -z "$p1" ] || [ -z "$p2" ]; then
        echo -e "${RED}Error: Two profile files required${NC}"
        echo "Usage: $0 compare <profile1> <profile2>"
        exit 1
    fi

    if [ ! -f "$p1" ] || [ ! -f "$p2" ]; then
        echo -e "${RED}Error: Profile file(s) not found${NC}"
        exit 1
    fi

    echo -e "${GREEN}Comparing profiles:${NC}"
    echo -e "  Base:   $p1"
    echo -e "  Current: $p2"
    echo ""

    # Show difference
    echo -e "${BLUE}Memory allocation differences:${NC}"
    go tool pprof -base="$p1" -top "$p2" 2>/dev/null | head -20
    echo ""

    echo -e "${BLUE}To open interactive comparison:${NC}"
    echo -e "${YELLOW}  go tool pprof -base=$p1 -http=:8080 $p2${NC}"
}

run_benchmarks() {
    echo -e "${GREEN}Running benchmarks with memory profiling...${NC}"
    echo ""

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local mem_profile="$PROFILE_DIR/bench_mem_${timestamp}.prof"

    cd "$PROJECT_ROOT"

    # Run benchmarks with memory profiling
    if go test ./pkg/... -bench=. -benchmem -memprofile="$mem_profile" -run=^$ | tee "$PROFILE_DIR/bench_${timestamp}.txt"; then
        echo ""
        echo -e "${GREEN}✓ Benchmark results saved${NC}"
        echo -e "${GREEN}✓ Memory profile: $mem_profile${NC}"
        echo -e "${GREEN}✓ Text output: $PROFILE_DIR/bench_${timestamp}.txt${NC}"
    else
        echo -e "${RED}✗ Benchmark profiling failed${NC}"
        return 1
    fi
}

run_all_checks() {
    echo -e "${GREEN}Running comprehensive memory checks...${NC}"
    echo ""

    local failed=0

    # Run leak tests
    echo -e "${BLUE}[1/2] Memory leak detection tests${NC}"
    if ! run_leak_tests; then
        failed=1
    fi
    echo ""

    # Run benchmarks
    echo -e "${BLUE}[2/2] Memory benchmarks${NC}"
    if ! run_benchmarks; then
        failed=1
    fi
    echo ""

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}✓ All memory checks passed${NC}"
        return 0
    else
        echo -e "${RED}✗ Some memory checks failed${NC}"
        return 1
    fi
}

clean_profiles() {
    echo -e "${YELLOW}Cleaning profile files...${NC}"

    if [ -d "$PROFILE_DIR" ]; then
        rm -rf "$PROFILE_DIR"/*
        echo -e "${GREEN}✓ Profile directory cleaned${NC}"
    else
        echo -e "${YELLOW}No profile directory found${NC}"
    fi
}

# Main script logic
case "${1:-help}" in
    test)
        print_header
        run_leak_tests
        ;;
    profile)
        print_header
        run_profile "${2}"
        ;;
    heap)
        print_header
        run_heap_profile "${2}"
        ;;
    analyze)
        print_header
        analyze_profile "${2}"
        ;;
    compare)
        print_header
        compare_profiles "${2}" "${3}"
        ;;
    benchmark)
        print_header
        run_benchmarks
        ;;
    check)
        print_header
        run_all_checks
        ;;
    clean)
        print_header
        clean_profiles
        ;;
    help|--help|-h)
        print_header
        print_usage
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        print_usage
        exit 1
        ;;
esac
