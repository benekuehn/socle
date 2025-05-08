import HeroSection from "@/components/HeroSection";
import { TerminalDemo } from "@/components/TerminalDemo";
import { KeyCommandsSection } from "@/components/KeyCommandsSection";
import { CtaSection } from "@/components/CtaSection";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-zinc-950 text-zinc-400">
      <main className="flex-1">
        <HeroSection />
        <TerminalDemo />
        <KeyCommandsSection />
        <CtaSection />
      </main>
    </div>
  );
}
