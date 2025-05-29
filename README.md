# Socle
Socle is the CLI tool purpose-built for managing stacked Git branches on GitHub, fostering focused pull requests, enabling better reviews, and keeping you in a state of productive flow.

## What are Stacked Branches?

Stacked branches (also known as stacked diffs) are a powerful way to break down complex features into manageable, reviewable units. Instead of one massive branch with dozens of commits or hundreds of lines, you create a sequence of smaller, dependent branches. Each branch builds upon the previous one, containing a distinct, logical piece of the larger feature.

[Learn more about stacked branches →](https://socle.dev/why-stacking-branches)

## Installation

```bash
brew install benekuehn/tap/socle
```

## Quick Start

### Create a new stack

```bash
so create my-feature
```

This creates a new branch and starts your stack. As you work on your feature, you can create additional stacked branches:

```bash
so create my-feature-part2
```

### View your stack

```bash
so log
```

This shows you the current state of your stack, including which branches are ready for review and which need work.

### Submit for review

```bash
so submit
```

This creates pull requests for all branches in your stack that are ready for review.

## Learn More

For detailed documentation and advanced usage, check out the [CLI documentation](./cli/README.md).

## Why Socle?

- **Simple & Intuitive**: Designed to get out of your way and enable a natural, productive flow
- **Open Source**: Fully open-source under the MIT license
- **No Enterprise Upselling**: Focused on making stacked branches accessible to everyone
- **Powerful Yet Uncomplicated**: Experience the full benefits of stacked changes through a tool that prioritizes clarity and ease of use

## License

MIT
