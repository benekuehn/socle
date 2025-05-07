import { useState } from "react";
import { motion } from "framer-motion";
import { Clipboard, Check } from "lucide-react";

export const BrewButton = () => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText("brew install benekuehn/socle/socle");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-center">
      <code>brew install benekuehn/socle/socle</code>
      <motion.button
        className="ml-3 cursor-pointer text-gray-400 hover:text-white transition-colors"
        onClick={handleCopy}
        whileTap={{ scale: 0.95 }}
        aria-label="Copy brew install command"
        onKeyPress={(e) => {
          if (e.key === "Enter" || e.key === " ") handleCopy();
        }}
      >
        {copied ? <Check className="text-green-600" /> : <Clipboard />}
      </motion.button>
    </div>
  );
};
