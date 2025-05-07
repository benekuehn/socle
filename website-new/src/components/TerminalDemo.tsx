'use client'

import React, { useRef, useState, useEffect } from 'react';
import { motion } from 'framer-motion';
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
  } = useTerminalAnimation();

  // Set terminal height only once on mount
  useEffect(() => {
    if (terminalRef.current && terminalHeight === undefined) {
      const rect = terminalRef.current.getBoundingClientRect();
      const available = window.innerHeight - rect.top - BOTTOM_OFFSET;
      setTerminalHeight(Math.max(MIN_HEIGHT, Math.min(available, MAX_HEIGHT)));
    }
  }, [terminalHeight]);

  return (
    <motion.div className="w-full max-w-4xl mx-auto" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.6, delay: 0.4 }}>
      <div className="text-center mb-6">
        <div className="inline-flex items-center px-4 py-2 bg-zinc-900/50 border border-zinc-800 rounded-full min-h-[2.5rem]">
          <span className="font-mono text-green-400">$ {typedCommand}</span>
        </div>
        <p className="mt-2 text-zinc-400 text-sm">Shows the sequence of tracked branches in your stack</p>
      </div>
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-r from-zinc-900/20 to-zinc-800/20 rounded-xl blur-3xl opacity-30"></div>
        <div
          ref={terminalRef}
          className="relative bg-[#0A0A0A] border border-zinc-800 rounded-xl overflow-hidden shadow-2xl"
          style={terminalHeight ? { height: terminalHeight, maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT } : { maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT }}
        >
          <div className="p-1 bg-[#111111]">
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