# `so` - Stacked Operations CLI

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

### Core Commands:
#### track
- Use this command when you are on a branch that you want to start tracking as part of a stack.
- It will prompt you to select the parent branch for the current branch.
- This stores the parent-child relationship and the stack's base branch in your local `.git/config`.

```bash
git checkout "<branch-name>" && so track #(select ⁠main as parent).
```

#### show
- Displays the current stack of branches you are on, based on the tracking information set by ⁠so track.
- Shows the lineage from the base branch up to your current branch, marking the current one.
- If the current branch isn't tracked, it will prompt you to use ⁠so track.

```bash
so show
```

#### create
- Creates a new branch stacked on top of your current tracked branch.
- Your current branch must be tracked first (use ⁠so track).
- If `[new-branch-name]` is omitted, you'll be prompted.
- If you have uncommitted changes, you'll be prompted for a commit message (or use `-m <message>`) and how to stage changes (`git add .` or `git add -p`).
- The changes will be committed on the new branch.
- Automatically tracks the new branch with the current branch as its parent.

```bash
⁠so create "<new-branch-name>" -m "<message>"
```

#### restack
- Rebases the entire current stack onto the latest version of its base branch (e.g., ⁠main).
- Requires Git >= 2.38 and uses `git rebase --update-refs.`
- Fetches the base branch from ⁠origin by default (use `--no-fetch` to skip).
- If conflicts occur, ⁠so will stop and instruct you to use standard `git rebase --continue` or `git rebase --abort` commands after resolving conflicts.

```bash
so restack
```

Getting Help:
You can get help for any command by using the `--help` flag:
```bash
so --help
so track --help
so create --help
```

### Configuration
so stores stack relationship metadata directly in your local repository's configuration file (⁠.git/config) using ⁠git config.
- branch.<branch-name>.socle-parent: The name of the parent branch in the stack.
- branch.<branch-name>.socle-base: The name of the ultimate base/trunk branch for the stack.

This information is local only and is not pushed to the remote repository.
(Future configuration options, like setting default base branches, might be added later.)
