"use client"

import { motion } from "framer-motion"
import { CommandBox } from "./CommandBox"
import { ArrowRight } from "lucide-react"
import { useRef, useEffect, useState } from "react"

const commands = [
  {
    command: "so create",
    description: "Creates a new branch stacked on top of the current branch, automatically tracking relationships."
  },
  {
    command: "so log",
    description: "Shows the sequence of tracked branches in your stack with status indicators."
  },
  {
    command: "so restack",
    description: "Updates your stack by rebasing each branch sequentially onto its updated parent."
  },
  {
    command: "so submit",
    description: "Pushes branches and creates or updates corresponding GitHub Pull Requests."
  }
]

export function KeyCommandsSection() {
  const sectionRef = useRef<HTMLElement>(null);
  const [sectionStyle, setSectionStyle] = useState<React.CSSProperties>({});

  useEffect(() => {
    const calculatePosition = () => {
      if (!sectionRef.current) return;

      const terminalElement = document.querySelector('[data-terminal-demo]');
      if (!terminalElement) return;

      const terminalRect = terminalElement.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const availableSpace = viewportHeight - terminalRect.bottom;
      const MIN_SPACE = 100; // Minimum space we want between terminal and commands
      const MAX_SPACE = 200; // Maximum space before we force above fold

      if (availableSpace < MIN_SPACE) {
        // Not enough space, position below fold
        setSectionStyle({
          marginTop: `${window.innerHeight - terminalRect.top + 16}px`
        });
      } else if (availableSpace > MAX_SPACE) {
        // Too much space, use normal margin
        setSectionStyle({
          marginTop: '4rem' // 64px, equivalent to py-16
        });
      } else {
        // Space is just right, use available space
        setSectionStyle({
          marginTop: `${availableSpace - 100}px` // 100px buffer
        });
      }
    };

    calculatePosition();
    window.addEventListener('resize', calculatePosition);
    return () => window.removeEventListener('resize', calculatePosition);
  }, []);

  return (
    <section 
      ref={sectionRef}
      id="commands" 
      className="container mx-auto px-4 py-16"
      style={sectionStyle}
    >
      <motion.h2
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="text-3xl font-bold text-center mb-12"
      >
        Key Commands
      </motion.h2>
      <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
        {commands.map((cmd, index) => (
          <CommandBox
            key={cmd.command}
            command={cmd.command}
            description={cmd.description}
            index={index}
          />
        ))}
      </div>
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5, delay: 0.2 }}
        className="text-center mt-10"
      >
        <a
          href="https://github.com/benekuehn/socle"
          className="inline-flex items-center text-zinc-400 hover:text-white transition-colors"
        >
          View all commands
          <ArrowRight className="ml-2 h-4 w-4" />
        </a>
      </motion.div>
    </section>
  )
} 