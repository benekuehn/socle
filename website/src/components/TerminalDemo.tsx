'use client'

import React, { useRef, useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useTerminalAnimation } from '@/hooks/useTerminalAnimation';
import { StaticTerminalOutput } from './StaticTerminalOutput';
import { SoLogOutput } from './SoLogOutput';
import { SoRestackOutput } from './SoRestackOutput';
import { Pause, Play } from 'lucide-react';

const MIN_HEIGHT = 320;
const MAX_HEIGHT = 600;
const BOTTOM_OFFSET = 80;


export const TerminalDemo: React.FC = () => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [terminalHeight, setTerminalHeight] = useState<number | undefined>(undefined);
  const {
    typedCommand,
    showOutput,
    showRestackOutput,
    clearing,
    isPlaying,
    handlePlayPause,
    prefersReducedMotion,
    currentCommand,
  } = useTerminalAnimation();

  // Set terminal height only once on mount
  useEffect(() => {
    if (terminalRef.current && terminalHeight === undefined) {
      const rect = terminalRef.current.getBoundingClientRect();
      const available = window.innerHeight - rect.top - BOTTOM_OFFSET;
      setTerminalHeight(Math.max(MIN_HEIGHT, Math.min(available, MAX_HEIGHT)));
    }
  }, [terminalHeight]);

  let commandExplanation = '';
  if (currentCommand === 'so restack') {
    commandExplanation = 'Rebase the current stack onto the latest base branch';
  } else {
    commandExplanation = 'Shows the sequence of tracked branches in your stack';
  }

  return (
    <motion.div className="w-full max-w-4xl mx-auto px-4 sm:px-6" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.6, delay: 0.4 }}>
      <div className="text-center mb-6">
        <div className="inline-flex items-center px-4 py-2 gap-2">
          <span>$</span><span className="font-mono text-zinc-50">{typedCommand}</span>
        </div>
        <AnimatePresence mode="wait" initial={false}>
          <motion.p
            key={currentCommand}
            className="mt-2 text-zinc-400 text-sm"
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.18 }}
          >
            {commandExplanation}
          </motion.p>
        </AnimatePresence>
      </div>
      <div className="relative">
        <div
          ref={terminalRef}
          data-terminal-demo
          className="relative rounded-xl overflow-hidden"
          style={{
            ...(terminalHeight ? { height: terminalHeight, maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT } : { maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT }),
            border: '2px solid transparent',
            borderRadius: '1rem',
            background: `linear-gradient(#0A0A0A, #0A0A0A) padding-box, radial-gradient(ellipse 80% 40% at 50% 0%, rgba(255,255,255,0.18) 0%, rgba(255,255,255,0.04) 60%, rgba(0,0,0,0) 100%) border-box`
          }}
        >
          <div className="p-1 bg-zinc-950">
            <div className="flex items-center px-4 py-2" style={{ minHeight: 28 }}>
              <div className="flex space-x-2">
                <div className="w-3 h-3 rounded-full bg-zinc-600"></div>
                <div className="w-3 h-3 rounded-full bg-zinc-600"></div>
                <div className="w-3 h-3 rounded-full bg-zinc-600"></div>
              </div>
              {!prefersReducedMotion && (
                <button
                  onClick={handlePlayPause}
                  className="ml-auto p-1 text-zinc-400 hover:text-white transition-colors focus:outline-none"
                  aria-label={isPlaying ? 'Pause animation' : 'Play animation'}
                >
                  {isPlaying ? (
                    <Pause className='w-4 h-4'/>
                    ) : (
                    <Play className='w-4 h-4'/>)}
                </button>
              )}
            </div>
          </div>
          <div className="p-6 font-mono text-sm text-left overflow-x-auto">
            {prefersReducedMotion ? (
              <StaticTerminalOutput />
            ) : (
              <>
                <div className="flex">
                  <span className="text-zinc-500 mr-2">$</span>
                  <span className="text-zinc-300">{typedCommand}</span>
                </div>
                {!clearing && (
                  <>
                    <SoLogOutput showOutput={showOutput} />
                    <SoRestackOutput showRestackOutput={showRestackOutput} />
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  );
}; 