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
        <span className="text-green-600">●</span> main
      </motion.div>
      <motion.div className="ml-4" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.2 }}>
        <span className="text-green-600">●</span> feature/auth
      </motion.div>
      <motion.div className="ml-8" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.3 }}>
        <span className="text-green-600">●</span> feature/login-form <span className="text-zinc-400">(current)</span>
      </motion.div>
      <motion.div className="ml-12" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.4 }}>
        <span className="text-yellow-600">○</span> feature/validation <span className="text-yellow-500">(needs rebase)</span>
      </motion.div>
    </div>
  );
}; 