'use client'

import React from 'react';
import { motion } from 'framer-motion';

interface SoLogOutputProps {
  showOutput: boolean;
}

export const SoLogOutput: React.FC<SoLogOutputProps> = ({ showOutput }) => {
  if (!showOutput) return null;

  return (
    <div className="mt-2">
      <motion.div initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.1 }}>
        <span className="text-green-600">●</span> <span className="text-zinc-400">○</span> <span className="text-white font-bold">feature/login-form</span> <span className="text-zinc-400">(up-to-date, no PR submitted)</span>
      </motion.div>
      <motion.div initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.2 }}>
        <span className="text-green-600">●</span> <span className="text-white">●</span> <span className="text-white font-bold">feature/auth</span> <span className="text-zinc-400">(up-to-date, pr opened)</span>
      </motion.div>
      <motion.div className="ml-9" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.3 }}>
        main <span className="text-zinc-400">(base)</span>
      </motion.div>
    </div>
  );
}; 