import { motion } from "framer-motion"

interface CommandBoxProps {
  command: string
  description: string
  index: number
}

export function CommandBox({ command, description, index }: CommandBoxProps) {
  const isEven = index % 2 === 0
  const xOffset = isEven ? -20 : 20

  return (
    <motion.div
      initial={{ opacity: 0, x: xOffset }}
      whileInView={{ opacity: 1, x: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      whileHover={{ y: -5, boxShadow: "0 0 20px 0 rgba(255, 255, 255, 0.05)" }}
      className="border border-zinc-800 rounded-lg p-6 bg-black/50 backdrop-blur-sm"
    >
      <h3 className="text-xl font-bold mb-2 flex items-center">
        <code className="bg-zinc-800 px-2 py-1 rounded mr-2 text-sm">{command}</code>
      </h3>
      <p className="text-zinc-400">{description}</p>
    </motion.div>
  )
} 