package ruff

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/horiagug/ruff-format-changes/internal/git"
)

// Ruff provides ruff formatting operations
type Ruff struct {
	dryRun   bool
	verbose  bool
	repoRoot string
}

// New creates a new Ruff instance
func New(repoRoot string, dryRun, verbose bool) *Ruff {
	return &Ruff{
		dryRun:   dryRun,
		verbose:  verbose,
		repoRoot: repoRoot,
	}
}

// CheckRuffInstalled verifies that ruff is installed and accessible
func CheckRuffInstalled() error {
	cmd := exec.Command("ruff", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ruff not found. Please install it with: pip install ruff")
	}
	return nil
}

// GetAbsolutePaths converts relative file paths to absolute paths
func (r *Ruff) GetAbsolutePaths(files []string) []string {
	var absolute []string
	for _, f := range files {
		absPath := filepath.Join(r.repoRoot, f)
		absolute = append(absolute, absPath)
	}
	return absolute
}

// FormatFilesByLineRanges runs ruff format on specific line ranges in files
func (r *Ruff) FormatFilesByLineRanges(fileChanges []git.FileChanges) error {
	if len(fileChanges) == 0 {
		if r.verbose {
			fmt.Println("No changed lines to format")
		}
		return nil
	}

	if r.verbose {
		fmt.Printf("Found %d Python file(s) with changed lines:\n", len(fileChanges))
		for _, fc := range fileChanges {
			fmt.Printf("  - %s\n", fc.FilePath)
			for _, lr := range fc.LineRanges {
				if lr.Start == lr.End {
					fmt.Printf("    Line %d\n", lr.Start)
				} else {
					fmt.Printf("    Lines %d-%d\n", lr.Start, lr.End)
				}
			}
		}
	}

	// Format each file with its changed line ranges
	for _, fc := range fileChanges {
		absPath := filepath.Join(r.repoRoot, fc.FilePath)

		for _, lineRange := range fc.LineRanges {
			if err := r.formatFileWithRange(absPath, lineRange); err != nil {
				return err
			}
		}
	}

	if !r.dryRun && r.verbose {
		fmt.Printf("\nSuccessfully formatted changed lines\n")
	}

	return nil
}

// formatFileWithRange formats a specific line range in a file
func (r *Ruff) formatFileWithRange(filePath string, lineRange git.LineRange) error {
	// Build ruff command with --range argument
	args := []string{"format"}

	if r.dryRun {
		args = append(args, "--check", "--diff")
	}

	// Add the range argument: --range 12:15 or --range 12 for single line
	// Format is start_line-end_line (1-based, as per ruff spec)
	rangeArg := formatRangeArg(lineRange.Start, lineRange.End)
	args = append(args, "--range", rangeArg)

	args = append(args, filePath)

	if r.verbose {
		fmt.Printf("Running: ruff %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("ruff", args...)
	cmd.Dir = r.repoRoot

	// Capture output
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		fmt.Println(string(output))
	}

	// ruff format returns 0 on success
	// ruff format --check returns 1 if files would be changed (this is expected)
	if err != nil && r.dryRun {
		// With --check, exit code 1 means files would be reformatted (expected behavior)
		if strings.Contains(string(output), "would be reformatted") ||
			strings.Contains(string(output), "would reformat") {
			return nil
		}
		// Check for actual ruff errors
		if strings.Contains(string(output), "error:") {
			return fmt.Errorf("ruff format failed: %w", err)
		}
		return nil
	} else if err != nil && !r.dryRun {
		// In non-dry-run mode, only some exit codes are errors
		if strings.Contains(string(output), "error:") {
			return fmt.Errorf("ruff format failed: %w", err)
		}
		return nil
	}

	return nil
}

// formatRangeArg formats the range argument for ruff format
// Returns format like "12:15" or "12" for single line
func formatRangeArg(start, end int) string {
	if start == end {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + ":" + strconv.Itoa(end)
}
