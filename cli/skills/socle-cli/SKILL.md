---
name: socle-cli
description: Use this skill when you need to guide or automate usage of the socle CLI (the `so` command) for stacked-branch workflows, including installing `so`, tracking branch stacks, navigating up/down/top/bottom, creating stacked branches, restacking, syncing, or submitting stacked PRs.
---

# Socle CLI

## Overview
Enable consistent, step-by-step guidance for using the `so` CLI to manage stacked Git branches and pull requests.

## Quick start
- Install `so` with Homebrew (`brew install benekuehn/tap/socle`).
- Verify installation with `so --version`.
- Work inside a Git repository before running stack commands.

## Core tasks

### Create and track a stack
- Create a new stacked branch from the current branch with `so create <branch-name>`.
- If there are uncommitted changes, pass `-m "commit message"` to include them on the new branch.
- Set stack relationships by running `so track` on each branch in order from the base branch upward.

### Navigate the stack
- Move one branch up or down: `so up`, `so down`.
- Jump to the base-adjacent or top branch: `so bottom`, `so top`.
- Review the current stack and rebase status: `so log`.

### Update a stack
- Rebase the stack onto updated parents with `so restack`.
- Use `--no-fetch` to skip fetching the remote base branch.
- Use `--force-push` or `--no-push` to control pushing rebased branches.

### Submit or sync PRs
- Create or update stacked PRs with `so submit` (requires GitHub auth via `GITHUB_TOKEN` or `gh auth login`).
- Use `--no-draft` to create non-draft PRs and `--force` to force-push before submitting.
- Keep branches in sync with the remote using `so sync` (use `--no-restack` to skip restacking).

## Notes and gotchas
- Stack metadata is stored locally in `.git/config` under `branch.<branch-name>.socle-parent` and `branch.<branch-name>.socle-base`.
- `so` commands assume the stack is tracked; run `so track` first if navigation or restack commands fail.
- If `so restack` stops due to conflicts, resolve via standard Git rebase commands, then rerun `so restack`.
- Keep stacks as independent as possible; avoid mixing unrelated work in the same stack.
- Keep each branch in a stack focused and small to reduce rebase conflicts and review overhead.
