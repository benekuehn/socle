import { motion } from "framer-motion"
import { BrewButton } from "../BrewButton";

export default function HeroSection() {
  return (
    <section className="container mx-auto px-4 pt-20 pb-8 md:pt-32 md:pb-10 flex flex-col items-center">
      <div className="max-w-3xl mx-auto text-center">
        <motion.h1
          className="text-4xl md:text-6xl font-bold tracking-tight mb-6"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          Effortless flow <br /> for stacked progress
        </motion.h1>
        <motion.p
          className="text-gray-400 mb-8"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1 }}
        >
          Socle is the CLI tool purpose-built for managing stacked Git branches on GitHub, fostering focused pull requests, enabling better reviews, and keeping you in a state of productive flow.</motion.p>
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-8">
          <div className="relative flex items-center bg-black border border-gray-800 rounded-lg px-6 py-3">
            <BrewButton />
          </div>
          <a
            href="https://github.com/benekuehn/socle"
            className="inline-flex items-center justify-center rounded-lg border border-gray-800 bg-black px-6 py-3 text-sm font-medium text-white hover:bg-gray-900 transition-colors"
          >
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  )
}
