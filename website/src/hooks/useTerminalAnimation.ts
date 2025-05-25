import { useEffect, useState, useRef } from "react";

// --- Constants for Animation Timing ---
const TYPING_INTERVAL = 100;
const POST_TYPING_DELAY = 500; // Delay after typing before showing output
const OUTPUT_DURATION = 2000; // How long output stays visible
const PREFERS_REDUCED_MOTION_INITIAL_DELAY = 50; // Short delay if any for reduced motion static content settle
const INITIAL_TERMINAL_SETTLE_DELAY = 1000; // New: Longer delay for terminal UI to animate in
const INTER_COMMAND_CLEAR_DELAY = 300; // New: Delay for blank terminal between commands
const NUM_ANIMATION_STEPS = 2; // 0 for "so log", 1 for "so restack"

// --- Types ---
interface PausedState {
    step: number;
    typedCommand: string;
    showOutput: boolean;       // For "so log" output
    showRestack: boolean;      // True if "so restack" command sequence is generally active
    showRestackOutput: boolean; // For "so restack" specific output visibility
}

interface AnimationCommandParams {
    commandStr: string;
    initialTypedCommand: string;
    initialOutputVisible: boolean;
    getIsCancelledOrPaused: () => boolean;
    setTypedCommandState: (cmd: string) => void;
    setOutputVisibleState: (visible: boolean) => void;
    setClearingState: (clearing: boolean) => void;
}

// --- Helper for Animating a Single Command ---
async function executeCommandAnimation(params: AnimationCommandParams): Promise<void> {
    const {
        commandStr,
        initialTypedCommand,
        initialOutputVisible,
        getIsCancelledOrPaused,
        setTypedCommandState,
        setOutputVisibleState,
        setClearingState,
    } = params;

    // 1. Type the command
    let currentTyped = initialTypedCommand;
    // Ensure the initial state is set if resuming mid-type or at start
    setTypedCommandState(currentTyped);

    if (currentTyped.length < commandStr.length) {
        await new Promise<void>(resolveTypingPromise => {
            const intervalId = setInterval(() => {
                if (getIsCancelledOrPaused()) {
                    clearInterval(intervalId);
                    return resolveTypingPromise();
                }
                const nextCharIndex = currentTyped.length;
                currentTyped += commandStr[nextCharIndex];
                setTypedCommandState(currentTyped);

                if (currentTyped.length === commandStr.length) {
                    clearInterval(intervalId);
                    resolveTypingPromise();
                }
            }, TYPING_INTERVAL);
        });
    }
    if (getIsCancelledOrPaused()) return;

    // 2. Delay, then show output (if not already visible from a paused state)
    if (!initialOutputVisible) {
        await new Promise(r => setTimeout(r, POST_TYPING_DELAY));
        if (getIsCancelledOrPaused()) return;
        setOutputVisibleState(true);
    } else {
        setOutputVisibleState(true); // Ensure it's set if resumed
    }
    if (getIsCancelledOrPaused()) return;

    // 3. Keep output visible for its duration
    await new Promise(r => setTimeout(r, OUTPUT_DURATION));
    if (getIsCancelledOrPaused()) return;

    // 4. Hide specific output and clear typed command.
    setOutputVisibleState(false);
    setTypedCommandState('');

    if (getIsCancelledOrPaused()) {
        // If paused here, handlePlayPause ensures clearing is false.
        return;
    }

    // 5. Briefly set terminal to "clearing" state for a blank appearance.
    setClearingState(true);
    await new Promise(r => setTimeout(r, INTER_COMMAND_CLEAR_DELAY));
    // Check for pause again, as handlePlayPause would have set clearing to false.
    // If not paused, ensure clearing is false before next command starts.
    if (!getIsCancelledOrPaused()) {
        setClearingState(false);
    }
    // If it was paused during the delay, handlePlayPause would have set clearing to false already.
}


export const useTerminalAnimation = () => {
    const [typedCommand, setTypedCommand] = useState('');
    const [showOutput, setShowOutput] = useState(false); // For "so log"
    const [showRestack, setShowRestack] = useState(false); // Overall active state for "so restack" step
    const [showRestackOutput, setShowRestackOutput] = useState(false); // For "so restack" output
    const [clearing, setClearing] = useState(false);
    const [isPlaying, setIsPlaying] = useState(true);
    const [initialDelayDone, setInitialDelayDone] = useState(false);

    const prefersReducedMotion = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const animationStepRef = useRef(0);
    const pausedStateRef = useRef<PausedState | null>(null);

    // Initial delay before any animation starts
    useEffect(() => {
        let delay = INITIAL_TERMINAL_SETTLE_DELAY;
        if (prefersReducedMotion) {
            // For reduced motion, content appears statically. A very short delay might be okay,
            // or setInitialDelayDone(true) directly if terminal entry animation is CSS only and doesn't interfere.
            delay = PREFERS_REDUCED_MOTION_INITIAL_DELAY; 
        }
        const timer = setTimeout(() => setInitialDelayDone(true), delay);
        return () => clearTimeout(timer);
    }, [prefersReducedMotion]); // Only re-run if prefersReducedMotion changes

    // Main Animation Loop
    useEffect(() => {
        if (!initialDelayDone || prefersReducedMotion || !isPlaying) {
            if (prefersReducedMotion) {
                setTypedCommand('so log');
                setShowOutput(true);
                setShowRestack(false);
                setShowRestackOutput(false);
                setClearing(false);
            }
            return; // Don't run loop if conditions not met
        }

        let cancelled = false;
        const getIsCancelledOrPaused = () => cancelled || !isPlaying;

        const runLoop = async () => {
            let consumedPausedData: PausedState | null = null;
            if (pausedStateRef.current && pausedStateRef.current.step === animationStepRef.current) {
                consumedPausedData = { ...pausedStateRef.current };
                pausedStateRef.current = null; // Consume the global paused state
            }

            while (!getIsCancelledOrPaused()) {
                const currentStep = animationStepRef.current;
                let commandParams: Omit<AnimationCommandParams, 'getIsCancelledOrPaused' | 'setTypedCommandState'>;

                if (currentStep === 0) { // "so log"
                    setShowRestack(false); // Ensure other step's active state is off

                    commandParams = {
                        commandStr: 'so log',
                        initialTypedCommand: consumedPausedData?.step === 0 ? consumedPausedData.typedCommand : '',
                        initialOutputVisible: consumedPausedData?.step === 0 ? consumedPausedData.showOutput : false,
                        setOutputVisibleState: setShowOutput,
                        setClearingState: setClearing,
                    };
                } else { // "so restack" (currentStep === 1)
                    // Set "so restack" step as active, unless resuming into an already active state
                    if (consumedPausedData?.step === 1) {
                        setShowRestack(consumedPausedData.showRestack);
                    } else {
                        setShowRestack(true);
                    }

                    commandParams = {
                        commandStr: 'so restack',
                        initialTypedCommand: consumedPausedData?.step === 1 ? consumedPausedData.typedCommand : '',
                        initialOutputVisible: consumedPausedData?.step === 1 ? consumedPausedData.showRestackOutput : false,
                        setOutputVisibleState: setShowRestackOutput,
                        setClearingState: setClearing,
                    };
                }

                await executeCommandAnimation({
                    ...commandParams,
                    getIsCancelledOrPaused,
                    setTypedCommandState: setTypedCommand,
                });

                if (getIsCancelledOrPaused()) break; // Exit loop if paused/cancelled during command animation

                // Clean up for the completed step
                if (currentStep === 1) {
                    setShowRestack(false); // Mark "so restack" step as no longer active
                }
                
                consumedPausedData = null; // Ensure paused data is only used for the first iteration after resume

                animationStepRef.current = (currentStep + 1) % NUM_ANIMATION_STEPS;
            }
        };

        runLoop();

        return () => {
            cancelled = true;
        };
    }, [isPlaying, prefersReducedMotion, initialDelayDone]);

    const handlePlayPause = () => {
        setIsPlaying(prevIsPlaying => {
            const newIsPlaying = !prevIsPlaying;
            if (!newIsPlaying) { // Transitioning to PAUSED
                pausedStateRef.current = {
                    step: animationStepRef.current,
                    typedCommand,
                    showOutput,
                    showRestack,
                    showRestackOutput,
                };
                setClearing(false); // Ensure terminal is not stuck in a cleared state on pause
            }
            // When transitioning to PLAYING, the useEffect for the loop will pick up.
            return newIsPlaying;
        });
    };

    return {
        typedCommand,
        showOutput,
        // showRestack is an internal management state now, not directly needed by TerminalDemo.tsx for rendering output.
        // If TerminalDemo needs to know if the "restack" command phase is active for styling the prompt differently,
        // we could expose a derived boolean like `isRestackCommandActive = isPlaying && animationStepRef.current === 1`.
        // For now, let's stick to what's directly used by the output components.
        showRestackOutput,
        clearing,
        isPlaying,
        handlePlayPause,
        prefersReducedMotion,
        currentCommand: prefersReducedMotion ? 'so log' : (animationStepRef.current === 0 ? 'so log' : 'so restack'),
    };
};