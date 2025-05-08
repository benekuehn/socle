"use client"
import { motion } from "framer-motion"
import Link from "next/link"
import { BrewButton } from "./BrewButton"
import { LinkOutButton } from "./LinkOutButton"

export function CtaSection() {
  return (
    <section className="container mx-auto px-4 py-16">
      <motion.div
        className="max-w-3xl mx-auto bg-gradient-to-b from-zinc-900/50 to-black border border-zinc-800 rounded-lg p-8 text-center"
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
        viewport={{ once: true }}
        whileHover={{
          boxShadow: "0 0 30px 0 rgba(255, 255, 255, 0.05)",
        }}
      >
        <h2 className="text-2xl md:text-3xl font-bold mb-4">Ready to transform your Git workflow?</h2>
        <p className="text-zinc-400 mb-8">
          Start using socle today and experience a more efficient way to work with stacked branches.
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