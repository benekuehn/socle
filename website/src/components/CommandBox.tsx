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
      className="border border-zinc-800 rounded-lg p-6"
    >
      <h3 className="text-xl font-bold mb-2 flex items-center">
        <code className="bg-zinc-800 px-2 py-1 rounded mr-2 text-sm">{command}</code>
      </h3>
      <p className="text-zinc-400">{description}</p>
    </motion.div>
  )
} 