export const LinkOutButton = ({ href, children }: { href: string, children: React.ReactNode }) => {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className="rounded-lg border border-zinc-800 px-6 py-3 text-sm font-medium text-zinc-100 hover:bg-zinc-900 hover:border-zinc-900 transition-colors">
          
      {children}
    </a>
  )
}