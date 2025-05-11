import { ReactNode } from 'react';
import { CopyButton } from './CopyButton';

interface CodeBlockProps {
  children: ReactNode;
}

export function CodeBlock({ children }: CodeBlockProps) {
  return (
    <div className="relative group">
      <div className="absolute right-2 top-2">
        <CopyButton text={children?.toString() || ''} />
      </div>
      <div className="bg-zinc-100 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 p-4 rounded-lg font-mono text-sm whitespace-pre">
        {children}
      </div>
    </div>
  );
}

interface InlineCodeProps {
  children: ReactNode;
}

export function InlineCode({ children }: InlineCodeProps) {
  return (
    <code className="bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 px-1.5 py-0.5 rounded font-mono text-sm">
      {children}
    </code>
  );
} 