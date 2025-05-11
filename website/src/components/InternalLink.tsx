import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { ReactNode } from "react";

interface InternalLinkProps {
  href: string;
  className?: string;
  children: ReactNode;
}

export function InternalLink({ href, className = "", children }: InternalLinkProps) {
  return (
    <Link
      href={href}
      className={`text-zinc-400 hover:text-white transition-colors ${className}`}
    >
      <span
            className="inline-flex items-center"
          >
           {children} 
            <ArrowRight className="ml-2 mt-0.5 h-4 w-4 align-middle" />
          </span>
    </Link>
  );
} 