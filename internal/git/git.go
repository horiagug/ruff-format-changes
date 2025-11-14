package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// LineRange represents a range of line numbers in a file
type LineRange struct {
	Start int
	End   int
}

// FileChanges represents the changed lines in a specific file
type FileChanges struct {
	FilePath   string
	LineRanges []LineRange
}

// Git provides Git operations
type Git struct {
	repoRoot string
	verbose  bool
}

// New creates a new Git instance
func New(verbose bool) (*Git, error) {
	g := &Git{verbose: verbose}

	// Get the git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	g.repoRoot = strings.TrimSpace(string(output))
	return g, nil
}

// GetCurrentBranch returns the current branch name
func (g *Git) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetChangedFiles returns the list of changed files compared to base branch
func (g *Git) GetChangedFiles(baseBranch string) ([]string, error) {
	// First, try to get the merge base between current branch and base branch
	cmd := exec.Command("git", "diff", "--name-only", baseBranch)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(output) == 0 {
		if g.verbose {
			fmt.Println("No changed files found")
		}
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter for Python files
	var pyFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".py") {
			pyFiles = append(pyFiles, file)
		}
	}

	return pyFiles, nil
}

// GetRepoRoot returns the root directory of the repository
func (g *Git) GetRepoRoot() string {
	return g.repoRoot
}

// GetChangedLineRanges returns the changed line ranges for each Python file
func (g *Git) GetChangedLineRanges(baseBranch string) ([]FileChanges, error) {
	// Get the list of changed Python files first
	changedFiles, err := g.GetChangedFiles(baseBranch)
	if err != nil {
		return nil, err
	}

	if len(changedFiles) == 0 {
		if g.verbose {
			fmt.Println("No changed files found")
		}
		return []FileChanges{}, nil
	}

	var fileChangesList []FileChanges

	// For each file, get the diff and extract line ranges
	for _, file := range changedFiles {
		ranges, err := g.getFileLineRanges(baseBranch, file)
		if err != nil {
			if g.verbose {
				fmt.Printf("Warning: Could not get line ranges for %s: %v\n", file, err)
			}
			continue
		}

		if len(ranges) > 0 {
			fileChangesList = append(fileChangesList, FileChanges{
				FilePath:   file,
				LineRanges: ranges,
			})
		}
	}

	return fileChangesList, nil
}

// getFileLineRanges extracts the changed line ranges for a single file
func (g *Git) getFileLineRanges(baseBranch, filePath string) ([]LineRange, error) {
	// Use git diff to get the unified diff format
	cmd := exec.Command("git", "diff", baseBranch, "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff for %s: %w", filePath, err)
	}

	return parseUnifiedDiff(string(output))
}

// parseUnifiedDiff parses unified diff format and extracts changed line ranges
// The hunk headers look like: @@ -12,5 +12,7 @@
// This means starting at line 12 in the old file, 5 lines, and starting at line 12 in new file, 7 lines

func parseUnifiedDiff(diff string) ([]LineRange, error) {
	lines := strings.Split(diff, "\n")
	ranges := []LineRange{}

	// Regex to match hunk headers: @@ -old_start,old_count +new_start,new_count @@
	// We extract new_start (match[1]) to initialize the absolute line counter.
	hunkHeaderRegex := regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

	// State variables for tracking the current position in the new file
	var currentNewLine int   // The absolute line number in the new file (N_abs)
	var changeRangeStart int // Start line of the current continuous block of added lines (0 if inactive)

	// Helper function to finalize and record an active change range
	finalizeRange := func() {
		if changeRangeStart > 0 {
			// The end of the range is the line *before* the currentNewLine counter
			ranges = append(ranges, LineRange{Start: changeRangeStart, End: currentNewLine - 1})
			changeRangeStart = 0 // Reset the range state
		}
	}

	for _, line := range lines {
		// 1. Check if this is a hunk header
		if match := hunkHeaderRegex.FindStringSubmatch(line); match != nil {
			// Finalize any active range from the previous hunk body (safety measure)
			finalizeRange()

			// Extract and initialize the absolute line number for the new hunk
			newStartLine, err := strconv.Atoi(match[1])
			if err != nil {
				// Handle potential non-numeric start line, though unlikely for a valid diff
				return nil, err
			}
			currentNewLine = newStartLine
			changeRangeStart = 0

			continue
		}

		// Only process content lines once a hunk has been initialized
		if currentNewLine == 0 {
			continue // Skip leading headers or unexpected lines outside hunks
		}

		// 2. Process hunk body lines based on prefix [4, 3]
		if len(line) > 0 {
			prefix := line

			switch prefix {
			case "+": // Added Line
				// If a new continuous range is starting, bookmark the line number
				if changeRangeStart == 0 {
					changeRangeStart = currentNewLine
				}
				currentNewLine++

			case " ": // Context Line (Unchanged)
				finalizeRange()
				currentNewLine++

			case "-": // Deleted Line
				finalizeRange()

			default:
			}
		}
	}

	// 3. Final check: If the diff ends with an active range, finalize it
	finalizeRange()

	return ranges, nil
}
