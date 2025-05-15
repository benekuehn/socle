# `so` - A CLI for Stacked Branch Workflows

`so` is a command-line tool designed to simplify working with stacked Git branches and pull requests. It helps manage the relationships between dependent branches, automating common tasks like rebasing and creating sequential branches.
This tool makes stacked development workflows smoother by adding metadata locally to your Git repository configuration, complementing native Git commands.

## Installation

Using Homebrew (Recommended)
The easiest way to install ⁠so is through Homebrew:

```bash
# Add the tap repository
brew tap benekuehn/socle

# Install the package
brew install socle
```

Verifying Installation
After installation, verify that `so` is working correctly:

```bash
so --version
```

## Basic Usag

Most so commands need to be run from within a Git repository.

<!-- CLI_REFERENCE_START -->

_This section is auto-generated. Do not edit manually._

### so bottom

Navigates to the first branch stacked directly on top of the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the first branch after the base in the sequence leading to the top.

```
so bottom [flags]
```

```
  -h, --help   help for bottom
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so create

Creates a new branch stacked on top of the current branch.

If a [branch-name] is not provided, you will be prompted for one.

If there are uncommitted changes in the working directory:

-   They will be staged and committed onto the _new_ branch.
-   You must provide a commit message via the -m flag, or you will be prompted.

```
so create [branch-name] [flags]
```

```
  -h, --help             help for create
  -m, --message string   Commit message to use for uncommitted changes
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so down

Navigates one level down the stack towards the base branch.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate parent of the current branch.

```
so down [flags]
```

```
  -h, --help   help for down
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so log

Shows the sequence of tracked branches leading from the stack's base
branch to the current branch, based on metadata set by 'socle track'.
Includes status indicating if a branch needs rebasing onto its parent.

```
so log [flags]
```

```
  -h, --help   help for log
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so restack

Updates the current stack by rebasing each branch sequentially onto its updated parent.
Handles remote 'origin' automatically.

Process:

1. Checks for clean state & existing Git rebase.
2. Fetches the base branch from 'origin' (unless --no-fetch).
3. Rebases each branch in the stack onto the latest commit of its parent.
    - Skips branches that are already up-to-date.
4. If conflicts occur:
    - Stops and instructs you to use standard Git commands (status, add, rebase --continue / --abort).
    - Run 'so restack' again after resolving or aborting the Git rebase.
5. If successful:
    - Prompts to force-push updated branches to 'origin' (use --force-push or --no-push to skip prompt).

```
so restack [flags]
```

```
      --force-push   Force push rebased branches without prompting
  -h, --help         help for restack
      --no-fetch     Skip fetching the remote base branch
      --no-push      Do not push branches after successful rebase
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so submit

Pushes branches in the current stack to the remote ('origin' by default)
and creates or updates corresponding GitHub Pull Requests.

-   Requires GITHUB_TOKEN environment variable with 'repo' scope.
-   Reads PR templates from .github/ or root directory.
-   Creates Draft PRs by default (use --no-draft to override).
-   Stores PR numbers locally in '.git/config' for future updates.

```
so submit [flags]
```

```
      --force      Force push branches
  -h, --help       help for submit
      --no-draft   Create non-draft Pull Requests
      --no-push    Skip pushing branches to remote
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so top

Navigates to the highest branch in the current stack.

The stack is determined by the tracking information set via 'so track'.
This command finds the last branch in the sequence starting from the base branch.

```
so top [flags]
```

```
  -h, --help   help for top
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so track

Associates the current branch with a parent branch to define its position
within a stack. This allows 'socle show' to display the specific stack you are on.

```
so track [flags]
```

```
  -h, --help   help for track
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

---

### so up

Navigates one level up the stack towards the tip.

The stack is determined by the tracking information set via 'so track'.
This command finds the immediate descendent of the current branch.

```
so up [flags]
```

```
  -h, --help   help for up
```

### Options inherited from parent commands

```
      --debug   Enable debug logging output
```

<!-- CLI_REFERENCE_END -->

### Configuration

`so` stores stack relationship metadata directly in your local repository's configuration file (⁠.git/config) using ⁠git config.

-   branch.<branch-name>.socle-parent: The name of the parent branch in the stack.
-   branch.<branch-name>.socle-base: The name of the ultimate base/trunk branch for the stack.

This information is local only and is not pushed to the remote repository.
(Future configuration options, like setting default base branches, might be added later.)

## Contributing

### Setting up your Go Environment

1. Verify Go Installation:
   Check if Go is installed by running `go version`. If it's not installed, follow the instructions at https://golang.org/doc/install to install Go.
2. Configure Go Environment PATH:
   The `go` install command places compiled binaries in the ⁠bin subdirectory of your ⁠GOPATH, or the directory specified by the ⁠GOBIN environment variable. Add this directory to your shell's ⁠PATH:

```bash
# Find your Go bin directory
go env GOPATH GOBIN

# Add to your PATH (in ~/.zshrc, ~/.bashrc, etc.)
export PATH="$HOME/go/bin:$PATH"
```

### Development Workflow with Make

We use a Makefile to simplify the development process. Here's how to use it:

```bash
# Clone the repository
git clone https://github.com/benekuehn/socle.git
cd socle/cli/so

# Build and install a development version globally
make dev-install
```

The `make dev-install` command will: 1. Run all linters to check code quality 2. Execute all tests to verify functionality 3. Build a binary named ⁠so-dev 4. Install it to your Go binary path so it's globally accessible

You can then run your development build from anywhere:

```bash
so-dev [command]
```
