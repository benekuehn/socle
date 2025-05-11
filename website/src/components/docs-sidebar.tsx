'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { ThemeToggle } from './theme-toggle'

export function DocsSidebar() {
  const pathname = usePathname()

  const isActive = (path: string) => {
    return pathname === path
  }

  return (
    <div className="inline-block w-64 bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-lg p-6 m-8 transition-colors">
      <div className="flex flex-col gap-6">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100 mb-4">Documentation</h2>
          <nav className="space-y-1">
            <Link 
              href="/docs/getting-started"
              className={`flex items-center px-3 py-2 text-sm rounded-md transition-colors ${
                isActive('/docs/getting-started') 
                  ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-medium' 
                  : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-50 dark:hover:bg-zinc-800/50'
              }`}
            >
              Getting Started
            </Link>
            <Link 
              href="/docs/commands"
              className={`flex items-center px-3 py-2 text-sm rounded-md transition-colors ${
                isActive('/docs/commands') 
                  ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-medium' 
                  : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-50 dark:hover:bg-zinc-800/50'
              }`}
            >
              Commands
            </Link>
          </nav>
        </div>

        {/* Theme Toggle */}
        <div className="border-t border-zinc-200 dark:border-zinc-800 pt-6">
          <ThemeToggle />
        </div>
      </div>
    </div>
  )
} 