package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/horiagug/ruff-format-changes/internal/git"
	"github.com/horiagug/ruff-format-changes/internal/ruff"
	"github.com/spf13/cobra"
)

func main() {
	var (
		baseBranch string
		dryRun     bool
		verbose    bool
	)

	rootCmd := &cobra.Command{
		Use:   "ruff-format-changes",
		Short: "Format only the changed lines in your Git branch using ruff",
		Long: `ruff-format-changes is a utility that runs 'ruff format' only on the lines
that have changed in your current Git branch compared to a base branch (usually main or master).

This helps keep your code formatted without reformatting the entire codebase.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(baseBranch, dryRun, verbose)
		},
	}

	rootCmd.Flags().StringVar(&baseBranch, "base", "", "Base branch to compare against (default: main or master)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying files")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(baseBranch string, dryRun, verbose bool) error {
	if err := ruff.CheckRuffInstalled(); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Initializing Git repository...")
	}

	gitClient, err := git.New(verbose)
	if err != nil {
		return err
	}

	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Current branch: %s\n", currentBranch)
	}

	if baseBranch == "" {
		baseBranch = determineBaseBranch(gitClient)
		if verbose {
			fmt.Printf("Using base branch: %s\n", baseBranch)
		}
	}

	if verbose {
		fmt.Printf("Comparing against branch: %s\n", baseBranch)
		fmt.Println("Getting changed lines...")
	}

	fileChanges, err := gitClient.GetChangedLineRanges(baseBranch)
	if err != nil {
		return err
	}

	if len(fileChanges) == 0 {
		fmt.Println("No Python files with changed lines in this branch")
		return nil
	}

	if verbose {
		fmt.Println()
	}

	ruffClient := ruff.New(gitClient.GetRepoRoot(), dryRun, verbose)

	if dryRun {
		fmt.Println("Running ruff format in dry-run mode (--check --diff)...")
		fmt.Println()
	} else {
		fmt.Println("Running ruff format on changed lines...")
		fmt.Println()
	}

	return ruffClient.FormatFilesByLineRanges(fileChanges)
}

func determineBaseBranch(gitClient *git.Git) string {
	parentBranch := findParentBranch()
	if parentBranch != "" {
		return parentBranch
	}

	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	commonBranches := []string{"main", "master", "develop", "development"}

	for _, branch := range commonBranches {
		if branch == currentBranch {
			continue
		}
		if branchExists(branch) {
			return branch
		}
	}

	defaultBranch := getRemoteDefaultBranch()
	if defaultBranch != "" && defaultBranch != currentBranch && branchExists(defaultBranch) {
		return defaultBranch
	}

	return "main"
}

// findParentBranch finds the parent branch of the current branch using git show-branch
// by parsing the output to find the nearest ancestor branch.
func findParentBranch() string {
	cmd := exec.Command("git", "show-branch")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	currentBranch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	currentBranchName := strings.TrimSpace(string(currentBranch))

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Find the separator line between branch list and commit history
	separatorIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "-----") {
			separatorIdx = i
			break
		}
	}

	if separatorIdx == -1 {
		return ""
	}

	var closestParent string
	currentIndent := -1

	// Find the current branch's indentation
	for i := 0; i < separatorIdx; i++ {
		line := lines[i]

		indent := 0
		for j := 0; j < len(line); j++ {
			if line[j] == ' ' {
				indent++
			} else {
				break
			}
		}

		branchName := extractBranchName(line)
		if branchName == currentBranchName {
			currentIndent = indent
			break
		}
	}

	// Find the non-current branch with maximum indentation less than currentIndent
	maxIndent := -1
	for i := 0; i < separatorIdx; i++ {
		line := lines[i]

		indent := 0
		for j := 0; j < len(line); j++ {
			if line[j] == ' ' {
				indent++
			} else {
				break
			}
		}

		branchName := extractBranchName(line)
		if branchName == "" || branchName == currentBranchName {
			continue
		}

		if indent < currentIndent && indent > maxIndent {
			maxIndent = indent
			closestParent = branchName
		}
	}

	if closestParent == "" {
		for i := 0; i < separatorIdx; i++ {
			line := lines[i]
			branchName := extractBranchName(line)
			if branchName != "" && branchName != currentBranchName {
				return branchName
			}
		}
	}

	return closestParent
}

// extractBranchName extracts the branch name from a git show-branch output line
// It extracts text within [brackets] and removes any ^ or ~ markers.
func extractBranchName(line string) string {
	startIdx := strings.Index(line, "[")
	endIdx := strings.Index(line, "]")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return ""
	}

	branchInfo := line[startIdx+1 : endIdx]

	for i, char := range branchInfo {
		if char == '^' || char == '~' {
			return branchInfo[:i]
		}
	}

	return branchInfo
}

// branchExists checks if a branch exists locally
func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	err := cmd.Run()
	return err == nil
}

// getRemoteDefaultBranch gets the default branch from the remote origin.
func getRemoteDefaultBranch() string {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	ref := strings.TrimSpace(string(output))
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
