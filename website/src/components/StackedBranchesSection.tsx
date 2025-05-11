"use client"

import { motion, AnimatePresence } from "framer-motion"
import { useRef, useEffect, useState } from "react"
import Link from "next/link"
import { InternalLink } from "./InternalLink"

const paragraphs = [
  {
    title: "Unlock Continuous Flow, Enhance Your Focus",
    content: "Achieve continuous, productive flow. Stacking enables parallel work and review, eliminating waits."
  },
  {
    title: "Foster Clarity, Enable Better Reviews",
    content: "Foster clarity with focused PRs. Stacking enables insightful, efficient reviews with ease."
  },
  {
    title: "Deliver Value Sooner, Build Momentum",
    content: "Ship value incrementally. Stacking enables shared team momentum and confident, progressive achievements."
  }
]

const STEP_HEIGHT = 0.7; // as fraction of viewport height
const TOP_SPACER = 0.01;
const BOTTOM_SPACER = STEP_HEIGHT;
const BREAK = 60; // px, the break before the next paragraph takes over

const BaseSVG = () => (
  <svg width="392" height="256" viewBox="0 0 392 256" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M390.998 111.714L196 225.167L1.00098 111.714L196 0.575195L390.998 111.714Z" fill="url(#paint0_linear_1_123)" stroke="url(#paint1_linear_1_123)"/>
    <path d="M0 111.709L196 225.745V256L0 141.963V111.709Z" fill="url(#paint2_linear_1_123)"/>
    <path d="M196 256L392 141.964V111.709L196 225.745V256Z" fill="url(#paint3_linear_1_123)"/>
    <defs>
      <linearGradient id="paint0_linear_1_123" x1="196" y1="0" x2="196" y2="257" gradientUnits="userSpaceOnUse">
        <stop stopColor="#8C8C8C"/>
        <stop offset="1" stopColor="#2A2A2A"/>
      </linearGradient>
      <linearGradient id="paint1_linear_1_123" x1="196" y1="0" x2="196" y2="225.745" gradientUnits="userSpaceOnUse">
        <stop stopColor="#444444"/>
        <stop offset="0.153846" stopOpacity="0"/>
      </linearGradient>
      <linearGradient id="paint2_linear_1_123" x1="98" y1="111.709" x2="98" y2="256" gradientUnits="userSpaceOnUse">
        <stop stopColor="#252525"/>
        <stop offset="1" stopColor="#ACACAC"/>
      </linearGradient>
      <linearGradient id="paint3_linear_1_123" x1="196" y1="-3.51021e-05" x2="304.5" y2="190.5" gradientUnits="userSpaceOnUse">
        <stop stopColor="#8C8C8C"/>
        <stop offset="1" stopColor="#2A2A2A"/>
      </linearGradient>
    </defs>
  </svg>
)

const MiddleSVG = () => (
  <svg width="300" height="196" viewBox="0 0 300 196" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M298.331 85.1152L150 171.418L1.66766 85.1162L150 0.575195L298.331 85.1152Z" fill="url(#paint0_linear_1_127)" stroke="url(#paint1_linear_1_127)"/>
    <path d="M0.666626 85.1113L150 171.996V195.047L0.666626 108.162V85.1113Z" fill="url(#paint2_linear_1_127)"/>
    <path d="M150 195.047L299.333 108.163V85.1115L150 171.996V195.047Z" fill="url(#paint3_linear_1_127)"/>
    <defs>
      <linearGradient id="paint0_linear_1_127" x1="150" y1="0" x2="150" y2="257.5" gradientUnits="userSpaceOnUse">
        <stop stopColor="#9C9C9C"/>
        <stop offset="1" stopColor="#2A2A2A"/>
      </linearGradient>
      <linearGradient id="paint1_linear_1_127" x1="150" y1="0" x2="150" y2="171.997" gradientUnits="userSpaceOnUse">
        <stop stopColor="#444444"/>
        <stop offset="0.153846" stopOpacity="0"/>
      </linearGradient>
      <linearGradient id="paint2_linear_1_127" x1="75.3333" y1="85.1113" x2="75.3333" y2="195.047" gradientUnits="userSpaceOnUse">
        <stop stopColor="#252525"/>
        <stop offset="1" stopColor="#ACACAC"/>
      </linearGradient>
      <linearGradient id="paint3_linear_1_127" x1="268" y1="175.5" x2="150" y2="1.00007" gradientUnits="userSpaceOnUse">
        <stop stopColor="#2A2A2A"/>
        <stop offset="1" stopColor="#9C9C9C"/>
      </linearGradient>
    </defs>
  </svg>
)

const TopSVG = () => (
  <svg width="216" height="141" viewBox="0 0 216 141" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path d="M108 0L215.333 61.174L108 123.622L0.666687 61.174L108 0Z" fill="url(#paint0_linear_1_131)"/>
    <path d="M0.666687 61.1738L108 123.622V140.19L0.666687 77.7417V61.1738Z" fill="url(#paint1_linear_1_131)"/>
    <path d="M108 140.191L215.333 77.742V61.1741L108 123.623V140.191Z" fill="url(#paint2_linear_1_131)"/>
    <defs>
      <linearGradient id="paint0_linear_1_131" x1="108" y1="0" x2="108" y2="257" gradientUnits="userSpaceOnUse">
        <stop stopColor="#ACACAC"/>
        <stop offset="1" stopColor="#2A2A2A"/>
      </linearGradient>
      <linearGradient id="paint1_linear_1_131" x1="54.3334" y1="61.1738" x2="54.3334" y2="140.19" gradientUnits="userSpaceOnUse">
        <stop stopColor="#252525"/>
        <stop offset="1" stopColor="#ACACAC"/>
      </linearGradient>
      <linearGradient id="paint2_linear_1_131" x1="196.5" y1="181" x2="108.37" y2="0.709889" gradientUnits="userSpaceOnUse">
        <stop stopColor="#2A2A2A"/>
        <stop offset="1" stopColor="#ACACAC"/>
      </linearGradient>
    </defs>
  </svg>
)

export function StackedBranchesSection() {
  const sectionRef = useRef<HTMLDivElement>(null);
  const triggerRefs = paragraphs.map(() => useRef<HTMLDivElement>(null));
  const [visibleIdx, setVisibleIdx] = useState<number|null>(null);
  const [middleVisible, setMiddleVisible] = useState(false);

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

  useEffect(() => {
    if (visibleIdx !== null && visibleIdx >= 1 && !middleVisible) {
      setMiddleVisible(true);
    }
    if (visibleIdx !== null && visibleIdx < 1 && middleVisible) {
      setMiddleVisible(false);
    }
  }, [visibleIdx, middleVisible]);

  return (
    <section ref={sectionRef} className="relative w-full bg-zinc-950" >
      <div className="w-full flex justify-center pt-10 relative z-100 -mb-64">
        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="text-3xl font-bold text-center text-zinc-300 "
        >
          Why Stacked Branches?
        </motion.h2>
      </div>
      <div className="max-w-6xl mx-auto grid md:grid-cols-2 h-full">
        <div className="h-full flex flex-col">
          <div className="sticky top-0 h-screen flex flex-col items-center justify-center pointer-events-none z-10">
            <AnimatePresence mode="wait">
              {visibleIdx !== null && (
                <motion.div
                  key={visibleIdx}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.3 }}
                  className="space-y-4 text-center pointer-events-auto flex flex-col items-center mt-8"
                >
                  <h3 className="text-xl font-semibold text-zinc-200">{paragraphs[visibleIdx].title}</h3>
                  <p className="text-zinc-400 max-w-md mx-auto">{paragraphs[visibleIdx].content}</p>
                  <div style={{ height: '2.5rem', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    {visibleIdx === 2 && (
                      <motion.div>
                        <InternalLink href="/why-stacking-branches">
                          Learn more about stacking
                        </InternalLink>
                      </motion.div>
                    )}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
          <div style={{ height: `1vh` }} />
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
        <div className="h-full flex flex-col">
          <div className="sticky top-0 h-screen flex flex-col items-center justify-center">
            <div className="relative w-96 h-64 flex items-center justify-center">
              <motion.div
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
                className="absolute left-1/2 top-0 -translate-x-1/2"
                style={{ zIndex: 1 }}
              >
                <BaseSVG />
              </motion.div>
              <AnimatePresence>
                {middleVisible && (
                  <motion.div
                    key="middle"
                    initial={{ opacity: 0, y: -40 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -40 }}
                    transition={{ duration: 0.5, delay: 0.1 }}
                    className="absolute left-1/2 top-0 -translate-x-1/2"
                    style={{ zIndex: 2 }}
                  >
                    <MiddleSVG />
                  </motion.div>
                )}
              </AnimatePresence>
              <AnimatePresence>
                {visibleIdx !== null && visibleIdx >= 2 && (
                  <motion.div
                    key="top"
                    initial={{ opacity: 0, y: -50 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -50 }}
                    transition={{ duration: 0.5, delay: 0.2 }}
                    className="absolute left-1/2 top-0 -translate-x-1/2"
                    style={{ zIndex: 3 }}
                  >
                    <TopSVG />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
} 