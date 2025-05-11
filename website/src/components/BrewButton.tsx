"use client"
import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Clipboard, Check } from "lucide-react";
import { CopyButton } from "./CopyButton";

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
      <CopyButton text="brew install benekuehn/tap/socle" ariaLabel="Copy brew install command" />
    </div>
  );
};
