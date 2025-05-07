import Link from "next/link"
import { motion } from "framer-motion"
import Image from "next/image"

export default function Header() {
  return (
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
  )
} 