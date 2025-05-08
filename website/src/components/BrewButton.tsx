"use client"
import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Clipboard, Check } from "lucide-react";

export const BrewButton = () => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText("brew install benekuehn/tap/socle");
    setCopied(true);
    setTimeout(() => setCopied(false), 1000);
  };

  return (
    <div className="flex items-center text-sm text-zinc-100 bg-zinc-900 rounded-md px-3 py-2 gap-3">
      <code>brew install benekuehn/tap/socle</code>
      <motion.button
        className="p-2 cursor-pointer text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-md transition-colors duration-150 ease-in-out"
        onClick={handleCopy}
        whileTap={{ scale: 0.90 }}
        aria-label="Copy brew install command"
        onKeyPress={(e) => {
          if (e.key === "Enter" || e.key === " ") handleCopy();
        }}
      >
        <AnimatePresence mode="wait" initial={false}>
          {copied ? (
            <motion.div
              key="check"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              transition={{ duration: 0.15 }}
            >
              <Check className="text-lime-500 w-3 h-3" />
            </motion.div>
          ) : (
            <motion.div
              key="clipboard"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              transition={{ duration: 0.15 }}
            >
              <Clipboard className="w-3 h-3" />
            </motion.div>
          )}
        </AnimatePresence>
      </motion.button>
    </div>
  );
};
