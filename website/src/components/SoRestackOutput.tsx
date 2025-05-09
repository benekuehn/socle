'use client'

import React from 'react';
import { motion } from 'framer-motion';

interface SoRestackOutputProps {
  showRestackOutput: boolean;
}

export const SoRestackOutput: React.FC<SoRestackOutputProps> = ({ showRestackOutput }) => {
  if (!showRestackOutput) return null;

  return (
    <div className="mt-4">
      <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, delay: 0.1 }}>
        Rebasing feature/validation onto feature/login-form...
      </motion.div>
      <motion.div className="text-green-600 mt-1" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, delay: 0.5 }}>
        ✓ Stack is up to date!
      </motion.div>
    </div>
  );
}; 