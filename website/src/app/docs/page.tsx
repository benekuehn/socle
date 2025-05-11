import Link from 'next/link'

export default function DocsPage() {
  return (
    <div>
      <h1 className="text-3xl font-semibold text-zinc-900 dark:text-zinc-100 mb-8">Documentation</h1>
      
      <div className="grid gap-4">
        <Link 
          href="/docs/getting-started" 
          className="group p-6 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors"
        >
          <h2 className="text-xl font-medium text-zinc-900 dark:text-zinc-100 mb-2 group-hover:text-zinc-700 dark:group-hover:text-zinc-300">Getting Started</h2>
          <p className="text-zinc-600 dark:text-zinc-400">Learn how to install so and create your first stack of branches.</p>
        </Link>

        <Link 
          href="/docs/commands" 
          className="group p-6 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors"
        >
          <h2 className="text-xl font-medium text-zinc-900 dark:text-zinc-100 mb-2 group-hover:text-zinc-700 dark:group-hover:text-zinc-300">Commands</h2>
          <p className="text-zinc-600 dark:text-zinc-400">Complete reference of all so commands and their options.</p>
        </Link>
      </div>
    </div>
  )
} 