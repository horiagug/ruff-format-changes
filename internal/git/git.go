package git

import (
	"fmt"
	"os"
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

// GetChangedFiles returns the list of changed Python files compared to base branch,
// including both tracked changes and untracked files
func (g *Git) GetChangedFiles(baseBranch string) ([]string, error) {
	// Get tracked changes
	cmd := exec.Command("git", "diff", "--name-only", baseBranch)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	fileMap := make(map[string]bool) // Use map to avoid duplicates
	if len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range files {
			if strings.HasSuffix(file, ".py") {
				fileMap[file] = true
			}
		}
	}

	// Get untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err = cmd.Output()
	if err != nil {
		if g.verbose {
			fmt.Printf("Warning: could not get untracked files: %v\n", err)
		}
	} else if len(output) > 0 {
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range files {
			if strings.HasSuffix(file, ".py") {
				fileMap[file] = true
			}
		}
	}

	if len(fileMap) == 0 {
		if g.verbose {
			fmt.Println("No changed files found")
		}
		return []string{}, nil
	}

	// Convert map to slice
	var pyFiles []string
	for file := range fileMap {
		pyFiles = append(pyFiles, file)
	}

	return pyFiles, nil
}

// GetRepoRoot returns the root directory of the repository
func (g *Git) GetRepoRoot() string {
	return g.repoRoot
}

// GetChangedLineRanges returns the changed line ranges for each Python file
func (g *Git) GetChangedLineRanges(baseBranch string) ([]FileChanges, error) {
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

// isFileUntracked checks if a file is untracked (not in git index)
func (g *Git) isFileUntracked(filePath string) (bool, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", filePath)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// getFileLineCount returns the total number of lines in a file
func getFileLineCount(filePath string) (int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	lines := strings.Count(string(content), "\n")
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		lines++
	}
	return lines, nil
}

// getFileLineRanges extracts the changed line ranges for a single file using git diff
// For untracked files, returns the entire file range
func (g *Git) getFileLineRanges(baseBranch, filePath string) ([]LineRange, error) {
	// Check if file is untracked
	untracked, err := g.isFileUntracked(filePath)
	if err != nil {
		if g.verbose {
			fmt.Printf("Warning: Could not determine if %s is untracked: %v\n", filePath, err)
		}
	}

	if untracked {
		// For untracked files, format the entire file
		lineCount, err := getFileLineCount(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to count lines in %s: %w", filePath, err)
		}
		if lineCount == 0 {
			return []LineRange{}, nil
		}
		return []LineRange{{Start: 1, End: lineCount}}, nil
	}

	// For tracked files, use git diff to find changed lines
	cmd := exec.Command("git", "diff", baseBranch, "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff for %s: %w", filePath, err)
	}

	return parseUnifiedDiff(string(output))
}

// parseUnifiedDiff parses unified diff format and extracts changed line ranges.
// It identifies line ranges that contain additions in the new file.
func parseUnifiedDiff(diff string) ([]LineRange, error) {
	lines := strings.Split(diff, "\n")
	ranges := []LineRange{}

	hunkHeaderRegex := regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

	var currentNewLine int
	var changeRangeStart int

	finalizeRange := func() {
		if changeRangeStart > 0 {
			ranges = append(ranges, LineRange{Start: changeRangeStart, End: currentNewLine - 1})
			changeRangeStart = 0
		}
	}

	for _, line := range lines {
		if match := hunkHeaderRegex.FindStringSubmatch(line); match != nil {
			finalizeRange()

			newStartLine, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, err
			}
			currentNewLine = newStartLine
			changeRangeStart = 0

			continue
		}

		if currentNewLine == 0 {
			continue
		}

		if len(line) > 0 {
			prefix := line[0:1]

			switch prefix {
			case "+":
				if changeRangeStart == 0 {
					changeRangeStart = currentNewLine
				}
				currentNewLine++

			case " ":
				finalizeRange()
				currentNewLine++

			case "-":
				finalizeRange()

			default:
			}
		}
	}

	finalizeRange()
	return ranges, nil
}
