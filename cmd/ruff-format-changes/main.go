package main

import (
	"fmt"
	"os"

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
	// Check if ruff is installed
	if err := ruff.CheckRuffInstalled(); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Initializing Git repository...")
	}

	// Initialize Git
	gitClient, err := git.New(verbose)
	if err != nil {
		return err
	}

	// Get current branch
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Current branch: %s\n", currentBranch)
	}

	// Determine base branch if not specified
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

	// Get changed line ranges for each file
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

	// Format changed lines
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
	// Try "main" first, then "master"
	branches := []string{"main", "master"}

	for _, branch := range branches {
		cmd := os.Getenv("GIT_BRANCH")
		if cmd == "" {
			// Simple check: try to get remote branch
			// In production, you might want to be more sophisticated
			return branch
		}
	}

	return "main"
}
