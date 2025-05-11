import { CodeBlock, InlineCode } from '@/components/CodeBlock';

export default function GettingStartedPage() {
  return (
    <div className="max-w-4xl mx-auto py-8 px-4">
      <h1 className="text-4xl font-bold mb-8 text-zinc-900 dark:text-zinc-100">Getting Started</h1>

      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">Installation</h2>
        <p className="mb-4 text-zinc-700 dark:text-zinc-300">The easiest way to install so is through Homebrew:</p>
        <CodeBlock>
          {`# Add the tap repository
          brew tap benekuehn/socle

          # Install the package
          brew install socle`}
        </CodeBlock>
        <p className="mb-4 text-zinc-700 dark:text-zinc-300">After installation, verify that so is working correctly:</p>
        <CodeBlock>so --version</CodeBlock>
      </section>

      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">Creating Your First Stack</h2>
        <p className="mb-4 text-zinc-700 dark:text-zinc-300">Let's create a simple stack of branches. We'll create three branches that depend on each other:</p>
        
        <div className="space-y-6">
          <div>
            <h3 className="text-xl font-medium mb-2 text-zinc-900 dark:text-zinc-100">1. Create the first branch</h3>
            <CodeBlock>
              {`# Start from your main branch
              git checkout main

              # Create and switch to the first branch
              so create feature-1`}
            </CodeBlock>
          </div>

          <div>
            <h3 className="text-xl font-medium mb-2 text-zinc-900 dark:text-zinc-100">2. Create the second branch</h3>
            <CodeBlock>
              {`# Make some changes and commit them
              echo "Feature 1" > feature1.txt
              git add feature1.txt
              git commit -m "Add feature 1"

              # Create the second branch
              so create feature-2`}
            </CodeBlock>
          </div>

          <div>
            <h3 className="text-xl font-medium mb-2 text-zinc-900 dark:text-zinc-100">3. Create the third branch</h3>
            <CodeBlock>
              {`# Make some changes and commit them
              echo "Feature 2" > feature2.txt
              git add feature2.txt
              git commit -m "Add feature 2"

              # Create the third branch
              so create feature-3`}
            </CodeBlock>
          </div>
        </div>
      </section>

      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">Tracking and Submitting Your Stack</h2>
        <p className="mb-4 text-zinc-700 dark:text-zinc-300">Now that we have created our branches, we need to track their relationships and submit them:</p>

        <div className="space-y-6">
          <div>
            <h3 className="text-xl font-medium mb-2 text-zinc-900 dark:text-zinc-100">1. Track the relationships</h3>
            <CodeBlock>
              {`# Go to the first branch
              so bottom

              # Track it as being based on main
              so track

              # Go to the second branch
              so up

              # Track it as being based on feature-1
              so track

              # Go to the third branch
              so up

              # Track it as being based on feature-2
              so track`}
            </CodeBlock>
          </div>

          <div>
            <h3 className="text-xl font-medium mb-2 text-zinc-900 dark:text-zinc-100">2. Submit the stack</h3>
            <CodeBlock>
              {`# From any branch in the stack
              so submit`}
            </CodeBlock>
            <p className="mt-2 text-zinc-600 dark:text-zinc-400">This will create draft pull requests for each branch in your stack.</p>
          </div>
        </div>
      </section>

      <section>
        <h2 className="text-2xl font-semibold mb-4 text-zinc-900 dark:text-zinc-100">Next Steps</h2>
        <p className="mb-4 text-zinc-700 dark:text-zinc-300">Now that you've created your first stack, you can:</p>
        <ul className="list-disc list-inside space-y-2 text-zinc-600 dark:text-zinc-400">
          <li>Use <InlineCode>so log</InlineCode> to see your stack</li>
          <li>Use <InlineCode>so up</InlineCode> and <InlineCode>so down</InlineCode> to navigate between branches</li>
          <li>Use <InlineCode>so restack</InlineCode> to update your stack when the base branch changes</li>
        </ul>
      </section>
    </div>
  )
} 