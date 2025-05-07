import React, { useRef, useState, useEffect } from 'react';
import { motion } from 'framer-motion';

const MIN_HEIGHT = 320;
const MAX_HEIGHT = 600;
const BOTTOM_OFFSET = 80;

const useTerminalAnimation = () => {
  const [typedCommand, setTypedCommand] = useState('');
  const [showOutput, setShowOutput] = useState(false);
  const [showRestack, setShowRestack] = useState(false);
  const [showRestackOutput, setShowRestackOutput] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [isPlaying, setIsPlaying] = useState(true);
  const [initialDelayDone, setInitialDelayDone] = useState(false);
  const prefersReducedMotion = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // Initial delay before animation starts
  useEffect(() => {
    if (prefersReducedMotion) {
      setInitialDelayDone(true);
      return;
    }
    const timer = setTimeout(() => setInitialDelayDone(true), 400);
    return () => clearTimeout(timer);
  }, [prefersReducedMotion]);

  // Typing animation for terminal (looping)
  useEffect(() => {
    if (!initialDelayDone) return;
    if (prefersReducedMotion) {
      setTypedCommand('so log');
      setShowOutput(true);
      setShowRestack(false);
      setShowRestackOutput(false);
      setClearing(false);
      return;
    }
    if (!isPlaying) return;
    let cancelled = false;
    const runLoop = async () => {
      while (!cancelled) {
        // so log
        setShowRestack(false);
        setShowRestackOutput(false);
        setClearing(false);
        let i = 0;
        const command = 'so log';
        await new Promise<void>(resolve => {
          const typing = setInterval(() => {
            setTypedCommand(command.substring(0, i));
            i++;
            if (i > command.length) {
              clearInterval(typing);
              setTimeout(() => {
                setShowOutput(true);
                setTimeout(() => {
                  setShowOutput(false);
                  setTypedCommand('');
                  setClearing(true);
                  setTimeout(() => {
                    setClearing(false);
                    // so restack
                    let j = 0;
                    const restackCommand = 'so restack';
                    setShowRestack(false);
                    setShowRestackOutput(false);
                    const typingRestack = setInterval(() => {
                      setShowRestack(true);
                      setTypedCommand(restackCommand.substring(0, j));
                      j++;
                      if (j > restackCommand.length) {
                        clearInterval(typingRestack);
                        setTimeout(() => {
                          setShowRestackOutput(true);
                          setTimeout(() => {
                            setShowRestackOutput(false);
                            setShowRestack(false);
                            setTypedCommand('');
                            setClearing(true);
                            setTimeout(() => {
                              setClearing(false);
                              resolve();
                            }, 400);
                          }, 2000);
                        }, 500);
                      }
                    }, 100);
                  }, 400);
                }, 2000);
              }, 500);
            }
          }, 100);
        });
      }
    };
    runLoop();
    return () => {
      cancelled = true;
    };
  }, [isPlaying, prefersReducedMotion, initialDelayDone]);

  const handlePlayPause = () => setIsPlaying(prev => !prev);

  return {
    typedCommand,
    showOutput,
    showRestack,
    showRestackOutput,
    clearing,
    isPlaying,
    handlePlayPause,
    prefersReducedMotion,
  };
};

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
        <div className="inline-flex items-center px-4 py-2 bg-gray-900/50 border border-gray-800 rounded-full min-h-[2.5rem]">
          <span className="font-mono text-green-400">$ {typedCommand}</span>
        </div>
        <p className="mt-2 text-gray-400 text-sm">Shows the sequence of tracked branches in your stack</p>
      </div>
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-r from-gray-900/20 to-gray-800/20 rounded-xl blur-3xl opacity-30"></div>
        <div
          ref={terminalRef}
          className="relative bg-[#0A0A0A] border border-gray-800 rounded-xl overflow-hidden shadow-2xl"
          style={terminalHeight ? { height: terminalHeight, maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT } : { maxHeight: MAX_HEIGHT, minHeight: MIN_HEIGHT }}
        >
          <div className="p-1 bg-[#111111]">
            <div className="flex items-center px-4 py-2" style={{ minHeight: 28 }}>
              <div className="flex space-x-2">
                <div className="w-3 h-3 rounded-full bg-gray-600"></div>
                <div className="w-3 h-3 rounded-full bg-gray-600"></div>
                <div className="w-3 h-3 rounded-full bg-gray-600"></div>
              </div>
              {!prefersReducedMotion && (
                <button
                  onClick={handlePlayPause}
                  className="ml-auto p-1 text-gray-400 hover:text-white transition-colors focus:outline-none"
                  aria-label={isPlaying ? 'Pause animation' : 'Play animation'}
                >
                  {isPlaying ? (
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><rect x="6" y="5" width="4" height="14" rx="1"/><rect x="14" y="5" width="4" height="14" rx="1"/></svg>
                  ) : (
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><polygon points="5,3 19,12 5,21 5,3"/></svg>
                  )}
                </button>
              )}
            </div>
          </div>
          <div className="p-6 font-mono text-sm text-left overflow-x-auto">
            {prefersReducedMotion ? (
              <>
                <div className="flex">
                  <span className="text-gray-500 mr-2">$</span>
                  <span className="text-gray-300">so log</span>
                </div>
                <div className="mt-2">
                  <span className="text-green-500">●</span> main
                  <br />
                  <span className="ml-4 text-green-500">●</span> feature/auth
                  <br />
                  <span className="ml-8 text-green-500">●</span> feature/login-form <span className="text-gray-400">(current)</span>
                  <br />
                  <span className="ml-12 text-yellow-500">○</span> feature/validation <span className="text-yellow-500">(needs rebase)</span>
                </div>
              </>
            ) : (
              !clearing && (
                <>
                  <div className="flex">
                    <span className="text-gray-500 mr-2">$</span>
                    <span className="text-gray-300">{typedCommand}</span>
                  </div>
                  {showOutput && (
                    <div className="mt-2">
                      <motion.div initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.1 }}>
                        <span className="text-green-500">●</span> main
                      </motion.div>
                      <motion.div className="ml-4" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.2 }}>
                        <span className="text-green-500">●</span> feature/auth
                      </motion.div>
                      <motion.div className="ml-8" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.3 }}>
                        <span className="text-green-500">●</span> feature/login-form <span className="text-gray-400">(current)</span>
                      </motion.div>
                      <motion.div className="ml-12" initial={{ opacity: 0, x: -5 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3, delay: 0.4 }}>
                        <span className="text-yellow-500">○</span> feature/validation <span className="text-yellow-500">(needs rebase)</span>
                      </motion.div>
                    </div>
                  )}
                  {showRestackOutput && (
                    <div className="mt-4">
                      <motion.div className="text-gray-300" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, delay: 0.1 }}>
                        Rebasing feature/validation onto feature/login-form...
                      </motion.div>
                      <motion.div className="text-green-500 mt-1" initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3, delay: 0.5 }}>
                        ✓ Stack is up to date!
                      </motion.div>
                    </div>
                  )}
                </>
              )
            )}
          </div>
        </div>
      </div>
    </motion.div>
  );
}; 