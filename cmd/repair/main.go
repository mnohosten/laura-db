package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/repair"
)

const (
	version = "1.0.0"
)

func main() {
	// Define command-line flags
	dataDir := flag.String("data-dir", "./data", "Database data directory")
	collection := flag.String("collection", "", "Specific collection to repair (empty = all collections)")
	operation := flag.String("operation", "validate", "Operation: validate, repair, defragment")
	dryRun := flag.Bool("dry-run", false, "Dry run mode (validate only, don't make changes)")
	rebuildIndexes := flag.Bool("rebuild-indexes", false, "Rebuild all indexes from scratch")
	removeOrphans := flag.Bool("remove-orphans", true, "Remove orphaned index entries")
	addMissing := flag.Bool("add-missing", true, "Add missing index entries")
	conflictResolution := flag.String("conflict-resolution", "fail", "Unique conflict resolution: first, last, fail")
	verbose := flag.Bool("verbose", false, "Verbose output")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "LauraDB Repair Tool v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOperations:\n")
		fmt.Fprintf(os.Stderr, "  validate     - Validate database integrity without making changes\n")
		fmt.Fprintf(os.Stderr, "  repair       - Repair database issues (use with caution)\n")
		fmt.Fprintf(os.Stderr, "  defragment   - Defragment database to reclaim space\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Validate entire database\n")
		fmt.Fprintf(os.Stderr, "  %s -data-dir ./mydb -operation validate\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  # Validate specific collection\n")
		fmt.Fprintf(os.Stderr, "  %s -data-dir ./mydb -collection users -operation validate\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  # Repair database in dry-run mode\n")
		fmt.Fprintf(os.Stderr, "  %s -data-dir ./mydb -operation repair -dry-run\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  # Repair with index rebuild\n")
		fmt.Fprintf(os.Stderr, "  %s -data-dir ./mydb -operation repair -rebuild-indexes\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  # Defragment database\n")
		fmt.Fprintf(os.Stderr, "  %s -data-dir ./mydb -operation defragment\n\n", filepath.Base(os.Args[0]))
	}

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("LauraDB Repair Tool v%s\n", version)
		os.Exit(0)
	}

	// Validate operation
	validOperations := map[string]bool{
		"validate":   true,
		"repair":     true,
		"defragment": true,
	}
	if !validOperations[*operation] {
		fmt.Fprintf(os.Stderr, "Error: Invalid operation '%s'. Must be one of: validate, repair, defragment\n", *operation)
		os.Exit(1)
	}

	// Print banner
	fmt.Printf("╔═══════════════════════════════════════════╗\n")
	fmt.Printf("║     LauraDB Repair Tool v%-16s║\n", version)
	fmt.Printf("╚═══════════════════════════════════════════╝\n\n")

	// Open database
	if *verbose {
		fmt.Printf("Opening database at: %s\n", *dataDir)
	}

	config := database.DefaultConfig(*dataDir)
	db, err := database.Open(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if *verbose {
		fmt.Printf("Database opened successfully\n\n")
	}

	// Execute operation
	switch *operation {
	case "validate":
		if err := runValidate(db, *collection, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "repair":
		opts := &repair.RepairOptions{
			RebuildIndexes:           *rebuildIndexes,
			RemoveOrphans:            *removeOrphans,
			AddMissingEntries:        *addMissing,
			UniqueConflictResolution: *conflictResolution,
			DryRun:                   *dryRun,
		}
		if err := runRepair(db, *collection, opts, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "defragment":
		if err := runDefragment(db, *collection, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("\n✓ Operation completed successfully\n")
}

func runValidate(db *database.Database, collectionName string, verbose bool) error {
	validator := repair.NewValidator(db)

	fmt.Printf("Running validation...\n")
	if collectionName != "" {
		fmt.Printf("Target: Collection '%s'\n\n", collectionName)
	} else {
		fmt.Printf("Target: Entire database\n\n")
	}

	var report *repair.ValidationReport
	var err error

	if collectionName != "" {
		report, err = validator.ValidateCollection(collectionName)
	} else {
		report, err = validator.Validate()
	}

	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Print summary
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Validation Results\n")
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Duration:     %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Collections:  %d\n", len(report.Collections))
	fmt.Printf("Documents:    %d\n", report.DocumentCount)
	fmt.Printf("Indexes:      %d\n", report.IndexCount)
	fmt.Printf("Health:       %s\n", healthStatus(report.IsHealthy))
	fmt.Printf("Issues:       %d\n", len(report.Issues))
	fmt.Printf("═══════════════════════════════════════════\n\n")

	// Print detailed summary
	fmt.Printf("%s\n\n", report.Summary())

	// Print issues if any
	if len(report.Issues) > 0 {
		fmt.Printf("Issues Found:\n")
		fmt.Printf("─────────────────────────────────────────\n")

		criticalIssues := 0
		warningIssues := 0
		infoIssues := 0

		for i, issue := range report.Issues {
			if verbose || issue.Severity == "critical" {
				fmt.Printf("\n[%d] %s\n", i+1, severityBadge(issue.Severity))
				fmt.Printf("    Type:        %s\n", issue.Type)
				fmt.Printf("    Collection:  %s\n", issue.Collection)
				if issue.DocumentID != "" {
					fmt.Printf("    Document ID: %s\n", issue.DocumentID)
				}
				if issue.IndexName != "" {
					fmt.Printf("    Index:       %s\n", issue.IndexName)
				}
				fmt.Printf("    Description: %s\n", issue.Description)
			}

			switch issue.Severity {
			case "critical":
				criticalIssues++
			case "warning":
				warningIssues++
			default:
				infoIssues++
			}
		}

		fmt.Printf("\n─────────────────────────────────────────\n")
		fmt.Printf("Issue Summary:\n")
		if criticalIssues > 0 {
			fmt.Printf("  ✗ Critical: %d\n", criticalIssues)
		}
		if warningIssues > 0 {
			fmt.Printf("  ⚠ Warnings: %d\n", warningIssues)
		}
		if infoIssues > 0 {
			fmt.Printf("  ℹ Info:     %d\n", infoIssues)
		}

		if !verbose && (warningIssues > 0 || infoIssues > 0) {
			fmt.Printf("\nUse -verbose flag to see all issues\n")
		}
	}

	return nil
}

func runRepair(db *database.Database, collectionName string, options *repair.RepairOptions, verbose bool) error {
	repairer := repair.NewRepairer(db)

	fmt.Printf("Running repair operation...\n")
	if collectionName != "" {
		fmt.Printf("Target: Collection '%s'\n", collectionName)
	} else {
		fmt.Printf("Target: Entire database\n")
	}

	if options.DryRun {
		fmt.Printf("Mode: DRY RUN (no changes will be made)\n")
	} else {
		fmt.Printf("Mode: LIVE (changes will be applied)\n")
	}

	fmt.Printf("\nRepair Options:\n")
	fmt.Printf("  Rebuild Indexes:       %v\n", options.RebuildIndexes)
	fmt.Printf("  Remove Orphans:        %v\n", options.RemoveOrphans)
	fmt.Printf("  Add Missing Entries:   %v\n", options.AddMissingEntries)
	fmt.Printf("  Conflict Resolution:   %s\n", options.UniqueConflictResolution)
	fmt.Printf("\n")

	var report *repair.RepairReport
	var err error

	if collectionName != "" {
		report, err = repairer.RepairCollection(collectionName, options)
	} else {
		report, err = repairer.Repair(options)
	}

	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}

	// Print summary
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Repair Results\n")
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Duration:     %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Issues Found: %d\n", len(report.Issues))
	fmt.Printf("Fixed:        %d\n", report.Fixed)
	fmt.Printf("Failed:       %d\n", report.Failed)
	fmt.Printf("═══════════════════════════════════════════\n\n")

	fmt.Printf("%s\n", report.Summary())

	// Print fixed issues if verbose
	if verbose && len(report.FixedIssues) > 0 {
		fmt.Printf("\nFixed Issues:\n")
		for i, issue := range report.FixedIssues {
			fmt.Printf("  [%d] %s - %s\n", i+1, issue.Type, issue.Description)
		}
	}

	// Print failed issues
	if len(report.FailedIssues) > 0 {
		fmt.Printf("\nFailed to Fix:\n")
		for i, issue := range report.FailedIssues {
			fmt.Printf("  [%d] %s - %s\n", i+1, issue.Type, issue.Description)
		}
	}

	return nil
}

func runDefragment(db *database.Database, collectionName string, verbose bool) error {
	defragmenter := repair.NewDefragmenter(db)

	fmt.Printf("Running defragmentation...\n")
	if collectionName != "" {
		fmt.Printf("Target: Collection '%s'\n\n", collectionName)
	} else {
		fmt.Printf("Target: Entire database\n\n")
	}

	var report *repair.DefragmentationReport
	var err error

	if collectionName != "" {
		report, err = defragmenter.DefragmentCollection(collectionName)
	} else {
		report, err = defragmenter.Defragment()
	}

	if err != nil {
		return fmt.Errorf("defragmentation failed: %w", err)
	}

	// Print summary
	percentSaved := 0.0
	if report.InitialFileSize > 0 {
		percentSaved = float64(report.SpaceSaved) / float64(report.InitialFileSize) * 100.0
	}

	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Defragmentation Results\n")
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Duration:          %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Pages Compacted:   %d\n", report.PagesCompacted)
	fmt.Printf("Initial Size:      %s\n", formatBytes(report.InitialFileSize))
	fmt.Printf("Final Size:        %s\n", formatBytes(report.FinalFileSize))
	fmt.Printf("Space Saved:       %s (%.2f%%)\n", formatBytes(report.SpaceSaved), percentSaved)
	fmt.Printf("Fragmentation:     %.2f%%\n", report.FragmentationRatio*100.0)
	fmt.Printf("═══════════════════════════════════════════\n\n")

	fmt.Printf("%s\n", report.Summary())

	return nil
}

// Helper functions

func healthStatus(isHealthy bool) string {
	if isHealthy {
		return "✓ Healthy"
	}
	return "✗ Issues Detected"
}

func severityBadge(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "✗ CRITICAL"
	case "warning":
		return "⚠ WARNING"
	default:
		return "ℹ INFO"
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
