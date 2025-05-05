# `so` - Stacked Branches CLI

`so` is a command-line interface tool designed to simplify working with stacked Git branches and pull requests. It helps manage the relationships between dependent branches, automating common tasks like rebasing and creating sequential branches.

This tool aims to make stacked development workflows smoother by adding metadata locally to your Git repository configuration, complementing native Git commands.

## Prerequisites

Before you can install and use `so`, you need the following installed on your system:

1.  **Go:** Version 1.18 or later is recommended.
    *   Installation instructions: [https://go.dev/doc/install](https://go.dev/doc/install)
    *   Verify installation: Run `go version` in your terminal.
2.  **Git:** Version **2.38** or later is **required** for the `restack` command's `--update-refs` functionality.
    *   Installation instructions: [https://git-scm.com/downloads](https://git-scm.com/downloads)
    *   Verify installation: Run `git --version` in your terminal.

## Installation and Setup

To run the `so` command from anywhere on your system, you need to compile it and place the binary in a directory included in your system's `PATH`. The standard Go way involves using `go install` and ensuring your Go binaries directory is in your `PATH`.

### Step 1: Verify Go Installation

```bash
go version
# Should output something like: go version go1.2x.y darwin/amd64
```

### Step 2: Configure Go Environment PATH
The ⁠go install command places compiled binaries in the ⁠bin subdirectory of your ⁠GOPATH, or the directory specified by the ⁠GOBIN environment variable if it's set. You need to add this directory to your shell's ⁠PATH variable.
1. Find your Go bin directory:
go env GOPATH GOBIN
- If GOBIN is set, that's your directory.
- If GOBIN is not set, your directory is ⁠$GOPATH/bin. Based on your previous output, this is likely `/Users/<your-username>/go/bin`. A common default is `$HOME/go/bin`.
2.	Add the directory to your PATH:
Edit your ⁠~/.zshrc file (e.g., `nano ~/.zshrc`). Add the following line (adjust the path if yours is different):
```bash
export PATH="/Users/benekuehn/go/bin:$PATH"
# Or use the general form:
# export PATH="$HOME/go/bin:$PATH"
```
3.	Apply the changes: Save the configuration file and either restart your terminal or run `source ~/.zshrc`.

### Step 3: Install so from Source
Navigate to the so CLI directory within the monorepo and run go install:
```bash
# Assuming you are in the root of the monorepo
cd cli
go install .
cd .. # Go back to root if needed
```
This command compiles the ⁠so tool and places the executable (so) into your Go bin directory (/Users/<your-username>/go/bin).

(Note: Once the tool is published, users could potentially install it directly using ⁠go install github.com/benekuehn/your-repo/cli@latest, but for development, install from your local source as shown above.)

### Step 4: Verify so Installation
Open a new terminal tab/window (to ensure the ⁠PATH changes are loaded) and run:
```bash
which so
# Should output the path, e.g.: /Users/benekuehn/go/bin/so

so --help
# Should display the main help message for the so command.
```

## Basic Usage
Most so commands need to be run from within a Git repository.

<!-- CLI_REFERENCE_START -->
*This section is auto-generated. Do not edit manually.*

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
  - They will be staged and committed onto the *new* branch.
  - You must provide a commit message via the -m flag, or you will be prompted.

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

- Requires GITHUB_TOKEN environment variable with 'repo' scope.
- Reads PR templates from .github/ or root directory.
- Creates Draft PRs by default (use --no-draft to override).
- Stores PR numbers locally in '.git/config' for future updates.

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
so stores stack relationship metadata directly in your local repository's configuration file (⁠.git/config) using ⁠git config.
- branch.<branch-name>.socle-parent: The name of the parent branch in the stack.
- branch.<branch-name>.socle-base: The name of the ultimate base/trunk branch for the stack.

This information is local only and is not pushed to the remote repository.
(Future configuration options, like setting default base branches, might be added later.)