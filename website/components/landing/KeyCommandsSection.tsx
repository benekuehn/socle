import { motion } from "framer-motion"

export default function KeyCommandsSection() {
  return (
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
        <a
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
        </a>
      </motion.div>
    </section>
  )
} 