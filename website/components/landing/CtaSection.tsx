import { motion } from "framer-motion"
import Link from "next/link"

export default function CtaSection({ copied, handleCopy }: { copied: boolean, handleCopy: () => void }) {
  return (
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
          <div className="relative flex items-center bg-black border border-gray-800 rounded-lg px-6 py-3">
            <code className="font-mono text-sm bg-gray-800 px-2 py-1 rounded">brew install benekuehn/socle/socle</code>
            <motion.span
              className="ml-3 cursor-pointer text-gray-400 hover:text-white transition-colors"
              onClick={handleCopy}
              whileTap={{ scale: 0.95 }}
              role="button"
              tabIndex={0}
              aria-label="Copy install command"
              onKeyPress={e => { if (e.key === 'Enter' || e.key === ' ') handleCopy(); }}
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
            </motion.span>
          </div>
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
  )
} 