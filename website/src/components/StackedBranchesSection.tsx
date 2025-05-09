"use client"

import { motion, AnimatePresence } from "framer-motion"
import { useRef, useEffect, useState } from "react"

const paragraphs = [
  {
    title: "Start with a Foundation",
    content: "Begin with your main branch as the foundation. This is where your stable, production-ready code lives."
  },
  {
    title: "Stack Your Features",
    content: "Create new branches stacked on top of each other. Each branch represents a focused feature or fix, building upon the work below it."
  },
  {
    title: "Maintain Order",
    content: "Keep your branches organized and up-to-date. Socle helps you manage the relationships between branches and ensures they stay in sync."
  }
]

const STEP_HEIGHT = 0.7; // as fraction of viewport height
const TOP_SPACER = 0.01;
const BOTTOM_SPACER = STEP_HEIGHT;
const BREAK = 60; // px, the break before the next paragraph takes over

export function StackedBranchesSection() {
  const sectionRef = useRef<HTMLDivElement>(null);
  const triggerRefs = paragraphs.map(() => useRef<HTMLDivElement>(null));
  const [visibleIdx, setVisibleIdx] = useState<number|null>(null);

  // Calculate total height for the stepper region
  const totalHeight = typeof window !== 'undefined'
    ? window.innerHeight * (TOP_SPACER + STEP_HEIGHT * paragraphs.length + BOTTOM_SPACER)
    : undefined;

  useEffect(() => {
    const handleScroll = () => {
      const center = window.innerHeight / 2;
      const offsets = triggerRefs.map(ref => {
        const rect = ref.current?.getBoundingClientRect();
        if (!rect) return 9999;
        return rect.top + rect.height / 2;
      });
      let found: number|null = null;
      for (let i = 0; i < offsets.length; i++) {
        const curr = offsets[i];
        const next = offsets[i + 1] ?? Infinity;
        // If we're in the break between two paragraphs, hide all
        if (i < offsets.length - 1 && Math.abs(next - center) < BREAK) {
          found = null;
          break;
        }
        // If this paragraph's trigger has passed the center, and the next hasn't yet, show this paragraph
        if (curr - center <= 0 && next - center > 0) {
          found = i;
          break;
        }
      }
      // Special case: before the first trigger, show the first paragraph (for scroll-in effect)
      if (offsets[0] - center > 0) {
        found = 0;
      }
      setVisibleIdx(found);
    };
    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll();
    return () => window.removeEventListener('scroll', handleScroll);
  }, [triggerRefs]);

  return (
    <section ref={sectionRef} className="relative w-full bg-zinc-950" style={totalHeight ? { height: totalHeight } : { minHeight: '100vh' }}>
      <div className="max-w-6xl mx-auto grid md:grid-cols-2 h-full">
        {/* Left column: sticky and centered for the whole region */}
        <div className="h-full flex flex-col">
          {/* Sticky, centered paragraph container (always present) */}
          <div className="sticky top-0 h-screen flex flex-col items-center justify-center pointer-events-none z-10">
            <AnimatePresence mode="wait">
              {visibleIdx !== null && (
                <motion.div
                  key={visibleIdx}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.3 }}
                  className="space-y-4 text-center pointer-events-auto"
                >
                  <h3 className="text-xl font-semibold text-zinc-200">{paragraphs[visibleIdx].title}</h3>
                  <p className="text-zinc-400 max-w-md mx-auto">{paragraphs[visibleIdx].content}</p>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
          {/* Spacers and triggers in normal flow after sticky container */}
          <div style={{ height: `1vh` }} />
          {/* First trigger is very short, second is a bit longer, rest are normal */}
          <div
            ref={triggerRefs[0]}
            style={{ height: `0.5vh` }}
            aria-hidden
          />
          <div
            ref={triggerRefs[1]}
            style={{ height: `40vh` }}
            aria-hidden
          />
          {paragraphs.slice(2).map((_, idx) => (
            <div
              key={idx + 2}
              ref={triggerRefs[idx + 2]}
              style={{ height: `${STEP_HEIGHT * 100}vh` }}
              aria-hidden
            />
          ))}
          <div style={{ height: `${BOTTOM_SPACER * 100}vh` }} />
        </div>
        {/* Right column: sticky and centered for the whole region */}
        <div className="h-full flex flex-col">
          <div className="sticky top-0 h-screen flex flex-col items-center justify-center">
            <motion.h2
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6 }}
              className="text-3xl font-bold text-center text-zinc-300 mb-10"
            >
              Why Stacked Branches?
            </motion.h2>
            <div className="relative w-64 h-96 flex items-center justify-center">
              <div className="absolute inset-0 bg-zinc-800 rounded-lg border border-zinc-700" />
              <div className="absolute inset-0 bg-zinc-700 rounded-lg border border-zinc-600" />
              <div className="absolute inset-0 bg-zinc-600 rounded-lg border border-zinc-500" />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
} 