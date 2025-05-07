"use client"
import { motion } from "framer-motion"
import { BrewButton } from "./BrewButton"


export default function HeroSection() {
  return (
    <section className="container mx-auto pt-20 pb-8 md:pt-32 md:pb-10 flex flex-col items-center">
      <div className="max-w-3xl text-center">
        <motion.h1
          className="text-4xl md:text-6xl font-bold tracking-tight mb-6 text-zinc-100"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          Effortless flow <br /> for stacked progress
        </motion.h1>
        <motion.p
          className="text-zinc-400 mb-8"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1 }}
        >
          Socle is the CLI tool purpose-built for managing stacked Git branches on GitHub, fostering focused pull requests, enabling better reviews, and keeping you in a state of productive flow.</motion.p>
        <motion.div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-8"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.15 }}>
          <BrewButton />
          <span>or</span>
          <a
            href="https://github.com/benekuehn/socle"
            className="rounded-lg border border-zinc-800 px-6 py-3 text-sm font-medium text-zinc-100 hover:bg-zinc-900 hover:border-zinc-900 transition-colors"
          >
            View on GitHub
          </a>
        </motion.div>
      </div>
    </section>
  )
}
