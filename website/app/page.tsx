"use client"

import { useEffect, useRef, useState } from "react"
import { motion } from "framer-motion"
import Link from "next/link"
import Image from "next/image"
import { TerminalDemo } from "../components/TerminalDemo"

export default function Home() {
  const [copied, setCopied] = useState(false)
  const [isPlaying, setIsPlaying] = useState(true)
  const animationRef = useRef<NodeJS.Timeout | null>(null)
  const terminalRef = useRef<HTMLDivElement>(null)
  const [terminalHeight, setTerminalHeight] = useState<number | undefined>(undefined)
  const prefersReducedMotion = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const [initialDelayDone, setInitialDelayDone] = useState(false)

  // Initial delay before animation starts
  useEffect(() => {
    if (prefersReducedMotion) {
      setInitialDelayDone(true)
      return
    }
    const timer = setTimeout(() => setInitialDelayDone(true), 400)
    return () => clearTimeout(timer)
  }, [prefersReducedMotion])

  // Dynamically set terminal height so its bottom is always 80px above the viewport bottom
  useEffect(() => {
    if (terminalRef.current && terminalHeight === undefined) {
      const rect = terminalRef.current.getBoundingClientRect()
      const available = window.innerHeight - rect.top - 80
      setTerminalHeight(Math.max(320, Math.min(available, 600)))
    }
  }, [terminalHeight])

  const handleCopy = () => {
    navigator.clipboard.writeText("brew install benekuehn/socle/socle")
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handlePlayPause = () => {
    setIsPlaying((prev) => !prev);
  }

  return (
    <div className="flex min-h-screen flex-col bg-black text-white">
      <header className="container mx-auto flex h-16 items-center justify-between px-4 border-b border-gray-800">
        <motion.div
          className="flex items-center space-x-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5 }}
        >
          <span className="text-xl font-medium tracking-tight">socle</span>
        </motion.div>
        <motion.nav
          className="flex items-center space-x-6"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
        >
          <Link href="#commands" className="text-sm text-gray-400 hover:text-white transition-colors">
            Commands
          </Link>
          <Link
            href="https://github.com/benekuehn/socle"
            className="text-sm text-gray-400 hover:text-white transition-colors"
          >
            GitHub
          </Link>
        </motion.nav>
      </header>

      <main className="flex-1">
        <section className="container mx-auto px-4 py-20 md:py-32 flex flex-col items-center">
          <div className="max-w-3xl mx-auto text-center">
            <motion.h1
              className="text-4xl md:text-6xl font-bold tracking-tight mb-6"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6 }}
            >
              Socle
            </motion.h1>

            <motion.p
              className="text-xl md:text-2xl text-gray-300 mb-8"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.1 }}
            >
              A lightweight CLI tool that helps you manage git branches as stacks.
            </motion.p>

            <motion.p
              className="text-gray-400 mb-8"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.2 }}
            >
              Socle: A short plinth or pedestal, often forming the base of a column, statue or art piece.
              <br />
              Connection: Represents the foundation upon which the stack is built. Simple, solid, and supportive.
              <br />
              Feel: Grounded, stable, understated strength.
            </motion.p>

            <motion.div
              className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-16"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.3 }}
            >
              <motion.div
                className="relative group"
                whileHover={{ scale: 1.02 }}
                transition={{ type: "spring", stiffness: 400, damping: 10 }}
              >
                <div className="absolute -inset-0.5 bg-gradient-to-r from-gray-600 to-gray-800 rounded-lg blur opacity-30 group-hover:opacity-100 transition duration-1000 group-hover:duration-200"></div>
                <div className="relative flex items-center bg-black border border-gray-800 rounded-lg px-6 py-3 font-mono text-sm">
                  brew install benekuehn/socle/socle
                  <motion.button
                    className="ml-3 text-gray-400 hover:text-white transition-colors"
                    onClick={handleCopy}
                    whileTap={{ scale: 0.95 }}
                  >
                    {copied ? (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="h-5 w-5 text-green-500"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="h-5 w-5"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                        />
                      </svg>
                    )}
                  </motion.button>
                </div>
              </motion.div>
              <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 400, damping: 10 }}>
                <Link
                  href="https://github.com/benekuehn/socle"
                  className="inline-flex items-center justify-center rounded-lg border border-gray-800 bg-black px-6 py-3 text-sm font-medium text-white hover:bg-gray-900 transition-colors"
                >
                  View on GitHub
                </Link>
              </motion.div>
            </motion.div>
          </div>

          {/* Command demonstration section */}
          <TerminalDemo />
        </section>

        <section id="commands" className="container mx-auto px-4 py-16 border-t border-gray-800">
          <motion.h2
            className="text-3xl font-bold text-center mb-12"
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            viewport={{ once: true }}
          >
            Key Commands
          </motion.h2>

          <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
            <motion.div
              className="border border-gray-800 rounded-lg p-6 bg-black/50 backdrop-blur-sm"
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5 }}
              viewport={{ once: true }}
              whileHover={{
                boxShadow: "0 0 20px 0 rgba(255, 255, 255, 0.05)",
                y: -5,
              }}
            >
              <h3 className="text-xl font-bold mb-2 flex items-center">
                <code className="bg-gray-800 px-2 py-1 rounded mr-2 text-sm">so create</code>
              </h3>
              <p className="text-gray-400">
                Creates a new branch stacked on top of the current branch, automatically tracking relationships.
              </p>
            </motion.div>

            <motion.div
              className="border border-gray-800 rounded-lg p-6 bg-black/50 backdrop-blur-sm"
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5 }}
              viewport={{ once: true }}
              whileHover={{
                boxShadow: "0 0 20px 0 rgba(255, 255, 255, 0.05)",
                y: -5,
              }}
            >
              <h3 className="text-xl font-bold mb-2 flex items-center">
                <code className="bg-gray-800 px-2 py-1 rounded mr-2 text-sm">so log</code>
              </h3>
              <p className="text-gray-400">
                Shows the sequence of tracked branches in your stack with status indicators.
              </p>
            </motion.div>

            <motion.div
              className="border border-gray-800 rounded-lg p-6 bg-black/50 backdrop-blur-sm"
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
              viewport={{ once: true }}
              whileHover={{
                boxShadow: "0 0 20px 0 rgba(255, 255, 255, 0.05)",
                y: -5,
              }}
            >
              <h3 className="text-xl font-bold mb-2 flex items-center">
                <code className="bg-gray-800 px-2 py-1 rounded mr-2 text-sm">so restack</code>
              </h3>
              <p className="text-gray-400">
                Updates your stack by rebasing each branch sequentially onto its updated parent.
              </p>
            </motion.div>

            <motion.div
              className="border border-gray-800 rounded-lg p-6 bg-black/50 backdrop-blur-sm"
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
              viewport={{ once: true }}
              whileHover={{
                boxShadow: "0 0 20px 0 rgba(255, 255, 255, 0.05)",
                y: -5,
              }}
            >
              <h3 className="text-xl font-bold mb-2 flex items-center">
                <code className="bg-gray-800 px-2 py-1 rounded mr-2 text-sm">so submit</code>
              </h3>
              <p className="text-gray-400">
                Pushes branches and creates or updates corresponding GitHub Pull Requests.
              </p>
            </motion.div>
          </div>

          <motion.div
            className="text-center mt-10"
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            viewport={{ once: true }}
          >
            <Link
              href="https://github.com/benekuehn/socle"
              className="inline-flex items-center text-gray-400 hover:text-white transition-colors"
            >
              View all commands
              <svg
                className="ml-2 h-4 w-4"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
              </svg>
            </Link>
          </motion.div>
        </section>

        <section className="container mx-auto px-4 py-16">
          <motion.div
            className="max-w-3xl mx-auto bg-gradient-to-b from-gray-900/50 to-black border border-gray-800 rounded-lg p-8 text-center"
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            viewport={{ once: true }}
            whileHover={{
              boxShadow: "0 0 30px 0 rgba(255, 255, 255, 0.05)",
            }}
          >
            <h2 className="text-2xl md:text-3xl font-bold mb-4">Ready to transform your Git workflow?</h2>
            <p className="text-gray-400 mb-8">
              Start using socle today and experience a more efficient way to work with stacked branches.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <motion.div
                className="relative flex items-center bg-black border border-gray-800 rounded-lg px-6 py-3 font-mono text-sm"
                whileHover={{ scale: 1.02 }}
                transition={{ type: "spring", stiffness: 400, damping: 10 }}
              >
                brew install benekuehn/socle/socle
                <motion.button
                  className="ml-3 text-gray-400 hover:text-white transition-colors"
                  onClick={handleCopy}
                  whileTap={{ scale: 0.95 }}
                >
                  {copied ? (
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="h-5 w-5 text-green-500"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  ) : (
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="h-5 w-5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                      />
                    </svg>
                  )}
                </motion.button>
              </motion.div>
              <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 400, damping: 10 }}>
                <Link
                  href="https://github.com/benekuehn/socle"
                  className="inline-flex items-center justify-center rounded-lg border border-gray-800 bg-black px-6 py-3 text-sm font-medium text-white hover:bg-gray-900 transition-colors"
                >
                  View Documentation
                </Link>
              </motion.div>
            </div>
          </motion.div>
        </section>
      </main>

      <footer className="border-t border-gray-800 py-8">
        <div className="container mx-auto px-4 flex flex-col md:flex-row justify-between items-center">
          <motion.div
            className="flex items-center space-x-2 mb-4 md:mb-0"
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            transition={{ duration: 0.5 }}
            viewport={{ once: true }}
          >
            <div className="relative h-6 w-6">
              <Image
                src="https://hebbkx1anhila5yf.public.blob.vercel-storage.com/Screenshot%202025-05-05%20at%2020.46.47%402x-Jel1zo78YGupjWujauReqZzrjBEEde.png"
                alt="Socle Logo"
                fill
                className="object-contain"
              />
            </div>
            <span className="font-bold">socle</span>
          </motion.div>
          <motion.div
            className="text-sm text-gray-500"
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            viewport={{ once: true }}
          >
            © {new Date().getFullYear()} socle. All rights reserved.
          </motion.div>
          <motion.div
            className="flex space-x-6 mt-4 md:mt-0"
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            viewport={{ once: true }}
          >
            <Link
              href="https://github.com/benekuehn/socle"
              className="text-gray-400 hover:text-white transition-colors"
            >
              GitHub
            </Link>
            <Link
              href="https://github.com/benekuehn/socle"
              className="text-gray-400 hover:text-white transition-colors"
            >
              Documentation
            </Link>
            <Link
              href="https://github.com/benekuehn/socle/issues"
              className="text-gray-400 hover:text-white transition-colors"
            >
              Issues
            </Link>
          </motion.div>
        </div>
      </footer>
    </div>
  )
}
