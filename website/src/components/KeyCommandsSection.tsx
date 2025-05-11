"use client"

import { motion } from "framer-motion"
import { CommandBox } from "./CommandBox"
import { ArrowRight } from "lucide-react"
import { InternalLink } from "./InternalLink"

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

  return (
    <section
      id="commands" 
      className="container mx-auto px-4 h-screen"
    >
      <motion.h2
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.6 }}
        className="text-3xl font-bold text-center mb-12 text-zinc-300"
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
        <InternalLink
          href="/docs/commands"
          className="inline-flex items-center"
        >
          View all commands
        </InternalLink>
      </motion.div>
    </section>
  )
} 