"use client"
import { motion } from "framer-motion"
import { BrewButton } from "./BrewButton"
import { LinkOutButton } from "./LinkOutButton"

export function CtaSection() {
  return (
    <section className="container mx-auto px-4 flex flex-col items-center justify-center text-center h-screen">
      <motion.div
       initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
        viewport={{ once: true }}
      >
        <h2 className="text-2xl md:text-3xl font-bold mb-4 text-zinc-300">Ready to transform your Git workflow?</h2>
        <p className="text-zinc-400 mb-8">
          Start using socle today and experience a more efficient way < br />to work with stacked branches on GitHub.
        </p>
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <BrewButton />
          <LinkOutButton href="https://github.com/benekuehn/socle">
            View on GitHub
          </LinkOutButton>
        </div>
      </motion.div>
    </section>
  )
} 