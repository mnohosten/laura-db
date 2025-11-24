#!/bin/bash

# LauraDB Code Coverage Report Generator
# Generates detailed coverage reports with summary

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== LauraDB Code Coverage Report ===${NC}"
echo ""

# Run tests with coverage
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test -v -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | tee test-output.log

# Check if tests passed
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    echo -e "${RED}Tests failed! Please fix failing tests before generating coverage report.${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}=== Package Coverage Breakdown ===${NC}"
echo ""

# Generate per-package coverage
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
go tool cover -func=coverage.out | grep -E "^github.com/mnohosten/laura-db/pkg" | while read line; do
    package=$(echo "$line" | awk '{print $1}' | sed 's/github.com\/mnohosten\/laura-db\///')
    coverage=$(echo "$line" | awk '{print $NF}')

    # Color code based on coverage
    coverage_num=$(echo "$coverage" | sed 's/%//')
    if (( $(echo "$coverage_num >= 80" | bc -l) )); then
        color=$GREEN
    elif (( $(echo "$coverage_num >= 60" | bc -l) )); then
        color=$YELLOW
    else
        color=$RED
    fi

    printf "  ${color}%-50s %6s${NC}\n" "$package" "$coverage"
done

echo ""
echo -e "${BLUE}=== Total Coverage ===${NC}"

# Get total coverage
total_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $NF}')
echo -e "  ${GREEN}$total_coverage${NC} of statements"

echo ""
echo -e "${BLUE}=== Coverage Files Generated ===${NC}"
echo "  - coverage.out (profile for tools)"
echo "  - coverage.html (HTML report)"
echo ""

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}✓ HTML coverage report generated${NC}"
echo -e "  Open coverage.html in your browser to view detailed coverage"

# Generate badge data
coverage_num=$(echo "$total_coverage" | sed 's/%//')
badge_color="green"
if (( $(echo "$coverage_num < 80" | bc -l) )); then
    badge_color="yellow"
fi
if (( $(echo "$coverage_num < 60" | bc -l) )); then
    badge_color="red"
fi

echo ""
echo -e "${BLUE}=== Coverage Badge ===${NC}"
echo "  ![Coverage](https://img.shields.io/badge/coverage-${total_coverage}-${badge_color})"
echo ""

# Cleanup
rm -f test-output.log

echo -e "${GREEN}✓ Coverage report complete${NC}"
