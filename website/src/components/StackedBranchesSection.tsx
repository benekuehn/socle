"use client"

import { motion } from "framer-motion"

const paragraphs = [
  {
    title: "Start with a Foundation",
    content: "Begin with your main branch as the foundation. This is where your stable, production-ready code lives."
  },
  {
    title: "Stack Your Features",
    content: "Create new branches stacked on top of each other. Each branch represents a focused feature or fix, building upon the work below it."
  },
  {
    title: "Maintain Order",
    content: "Keep your branches organized and up-to-date. Socle helps you manage the relationships between branches and ensures they stay in sync."
  }
]

export function StackedBranchesSection() {
  return (
    <section className="relative w-full bg-zinc-950">
      <div className="max-w-6xl mx-auto grid md:grid-cols-2 min-h-screen">
        {/* Scrolling left column with sticky paragraphs */}
        <div className="flex flex-col gap-0 w-full scroll-smooth snap-y snap-mandatory">
          {/* Spacer before first paragraph */}
          <div className="h-[50vh]" />
          {paragraphs.map((paragraph, idx) => (
            <section
              key={idx}
              className="h-[70vh] relative"
            >
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, amount: 0.7 }}
                transition={{ duration: 0.6 }}
                className="sticky top-1/2 -translate-y-1/2 space-y-4 text-center"
              >
                <h3 className="text-xl font-semibold text-zinc-200">{paragraph.title}</h3>
                <p className="text-zinc-400 max-w-md mx-auto">{paragraph.content}</p>
              </motion.div>
            </section>
          ))}
          {/* Spacer after last paragraph */}
          <div className="h-[40vh]" />
        </div>
        {/* Sticky right column */}
        <div className="flex flex-col items-center justify-center sticky top-0 h-screen">
          <motion.h2
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="text-3xl font-bold text-center text-zinc-300 mb-10"
          >
            Why Stacked Branches?
          </motion.h2>
          <div className="relative w-64 h-96 flex items-center justify-center">
            <div className="absolute inset-0 bg-zinc-800 rounded-lg border border-zinc-700" />
            <div className="absolute inset-0 bg-zinc-700 rounded-lg border border-zinc-600" />
            <div className="absolute inset-0 bg-zinc-600 rounded-lg border border-zinc-500" />
          </div>
        </div>
      </div>
    </section>
  )
} 