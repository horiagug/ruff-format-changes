# ruff-format-changes

A simple CLI utility that runs `ruff format` on only the **lines changed** in your Git branch.

## Purpose

When working on feature branches, you might want to format only the specific lines you've changed, without reformatting the entire codebase. This tool uses ruff's `--range` argument to format only the exact lines that differ from the base branch.

## Features

- Detects changed files in current Git branch (compared to main/master)
- Extracts the exact line ranges that were modified
- Runs `ruff format --range` only on those changed lines in each file
- Supports dry-run mode to preview changes
- Shows detailed output of what lines were formatted
- Configurable base branch (default: main/master)

## Installation

```bash
go install github.com/horiagug/ruff-format-changes@latest
```

Or build from source:

```bash
go build -o ruff-format-changes ./cmd/ruff-format-changes
```

## Usage

### Basic usage (format changed files in-place):

```bash
ruff-format-changes
```

### With options:

```bash
# Preview changes without modifying files
ruff-format-changes --dry-run

# Format against a specific base branch
ruff-format-changes --base develop

# Show verbose output
ruff-format-changes --verbose

# Combine options
ruff-format-changes --dry-run --base develop --verbose
```

## Options

- `--base string` - Base branch to compare against (default: "main" or "master")
- `--dry-run` - Preview changes without modifying files
- `--verbose` - Show detailed output
- `--help` - Show help message

## How it works

1. Detects your current Git branch
2. Identifies the base branch (configurable, defaults to main/master)
3. Gets the list of changed files using `git diff --name-only`
4. Filters for Python files (\*.py)
5. For each changed file, parses the unified diff to extract the exact line ranges that were modified
6. Runs `ruff format --range START-END` on each changed line range
7. Reports what was formatted

## Requirements

- Go 1.21+
- Git
- Python with ruff installed (`pip install ruff`)

## Development

```bash
# Run tests
go test ./...

# Build locally
go build -o ruff-format-changes ./cmd/ruff-format-changes

# Run locally
./ruff-format-changes --help
```

## License

MIT
