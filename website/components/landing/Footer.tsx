import Link from "next/link"
import { motion } from "framer-motion"
import Image from "next/image"

export default function Footer() {
  return (
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
  )
} 