import { DocsSidebar } from '@/components/docs-sidebar'

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="relative bg-white dark:bg-zinc-950 transition-colors min-h-screen">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex">
        <aside
          className="w-64 flex-shrink-0 sticky top-24 self-start"
          style={{ zIndex: 20 }}
        >
          <DocsSidebar />
        </aside>
        <main className="flex-1 min-w-0">
          <div className="max-w-4xl mx-auto px-6 py-8">
            <div className="prose dark:prose-invert max-w-none prose-pre:bg-zinc-900 dark:prose-pre:bg-zinc-900 prose-pre:text-zinc-100 dark:prose-pre:text-zinc-100 prose-code:text-zinc-900 dark:prose-code:text-zinc-100 prose-code:bg-zinc-100 dark:prose-code:bg-zinc-800 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:font-mono prose-code:text-sm">
              {children}
            </div>
          </div>
        </main>
      </div>
    </div>
  )
} 