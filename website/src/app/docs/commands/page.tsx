import { CodeBlock, InlineCode } from '@/components/CodeBlock';
import { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Docs: Overview of all Socle Commands',
  description: 'Learn what stacked branches are, their benefits for Git/GitHub workflows (focused PRs, faster reviews), and how Socle helps enable effortless flow.',
  openGraph: {
type: 'article',
    },
}

export default function CommandsPage() {
  return (
    <div className="max-w-4xl mx-auto py-8 px-4">
      <h1 className="text-4xl font-bold mb-8 text-zinc-900 dark:text-zinc-100">Command Reference</h1>

      <div className="space-y-12">
        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so bottom</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Navigates to the first branch stacked directly on top of the base branch.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">The stack is determined by the tracking information set via &apos;so track&apos; or &apos;so create&apos;. This command finds the first branch after the base in the sequence leading to the top.</p>
          <CodeBlock>so bottom [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for bottom</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so create</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Creates a new branch stacked on top of the current branch.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">If a [branch-name] is not provided, you will be prompted for one.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">If there are uncommitted changes in the working directory:</p>
          <ul className="list-disc list-inside mb-4 text-zinc-600 dark:text-zinc-400">
            <li>They will be staged and committed onto the new branch.</li>
            <li>You must provide a commit message via the -m flag, or you will be prompted.</li>
          </ul>
          <CodeBlock>so create [branch-name] [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for create</li>
              <li><InlineCode>-m, --message string</InlineCode> Commit message to use for uncommitted changes</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so down</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Navigates one level down the stack towards the base branch.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">The stack is determined by the tracking information set via &apos;so track&apos; or &apos;so create&apos;. This command finds the immediate parent of the current branch.</p>
          <CodeBlock>so down [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for down</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so log</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Shows the sequence of tracked branches leading from the stack&apos;s base branch to the current branch, based on metadata set by &apos;so track&apos; or &apos;so create&apos;.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Includes status indicating if a branch needs rebasing onto its parent.</p>
          <CodeBlock>so log [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for log</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so restack</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Updates the current stack by rebasing each branch sequentially onto its updated parent. Handles remote &apos;origin&apos; automatically.</p>
          <div className="mb-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Process:</h3>
            <ol className="list-decimal list-inside text-zinc-600 dark:text-zinc-400 space-y-2">
              <li>Checks for clean state & existing Git rebase.</li>
              <li>Fetches the base branch from &apos;origin&apos; (unless --no-fetch).</li>
              <li>Rebases each branch in the stack onto the latest commit of its parent.</li>
              <li>Skips branches that are already up-to-date.</li>
              <li>If conflicts occur:
                <ul className="list-disc list-inside ml-6 mt-2">
                  <li>Stops and instructs you to use standard Git commands (status, add, rebase --continue / --abort).</li>
                  <li>Run &apos;so restack&apos; again after resolving or aborting the Git rebase.</li>
                </ul>
              </li>
              <li>If successful:
                <ul className="list-disc list-inside ml-6 mt-2">
                  <li>Prompts to force-push updated branches to &apos;origin&apos; (use --force-push or --no-push to skip prompt).</li>
                </ul>
              </li>
            </ol>
          </div>
          <CodeBlock>so restack [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>--force-push</InlineCode> Force push rebased branches without prompting</li>
              <li><InlineCode>-h, --help</InlineCode> help for restack</li>
              <li><InlineCode>--no-fetch</InlineCode> Skip fetching the remote base branch</li>
              <li><InlineCode>--no-push</InlineCode> Do not push branches after successful rebase</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so submit</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Pushes branches in the current stack to the remote (&apos;origin&apos; by default) and creates or updates corresponding GitHub Pull Requests.</p>
          <ul className="list-disc list-inside mb-4 text-zinc-600 dark:text-zinc-400">
            <li>Requires GITHUB_TOKEN environment variable with &apos;repo&apos; scope.</li>
            <li>Reads PR templates from .github/ or root directory.</li>
            <li>Creates Draft PRs by default (use --no-draft to override).</li>
            <li>Stores PR numbers locally in &apos;.git/config&apos; for future updates.</li>
          </ul>
          <CodeBlock>so submit [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>--force</InlineCode> Force push branches</li>
              <li><InlineCode>-h, --help</InlineCode> help for submit</li>
              <li><InlineCode>--no-draft</InlineCode> Create non-draft Pull Requests</li>
              <li><InlineCode>--no-push</InlineCode> Skip pushing branches to remote</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so top</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Navigates to the highest branch in the current stack.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">The stack is determined by the tracking information set via &apos;so track&apos; or &apos;so create&apos;. This command finds the last branch in the sequence starting from the base branch.</p>
          <CodeBlock>so top [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for top</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so track</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Associates the current branch with a parent branch to define its position within a stack. This allows &apos;so log&apos; to display the specific stack you are on.</p>
          <CodeBlock>so track [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for track</li>
            </ul>
          </div>
        </section>

        <section>
          <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">so up</h2>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">Navigates one level up the stack towards the tip.</p>
          <p className="mb-4 text-zinc-700 dark:text-zinc-300">The stack is determined by the tracking information set via &apos;so track&apos; or &apos;so create&apos;. This command finds the immediate descendent of the current branch.</p>
          <CodeBlock>so up [flags]</CodeBlock>
          <div className="mt-4">
            <h3 className="text-lg font-medium mb-2 text-zinc-900 dark:text-zinc-100">Flags:</h3>
            <ul className="list-disc list-inside text-zinc-600 dark:text-zinc-400">
              <li><InlineCode>-h, --help</InlineCode> help for up</li>
            </ul>
          </div>
        </section>
      </div>
    </div>
  )
}
