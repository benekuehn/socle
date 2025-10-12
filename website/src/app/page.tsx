import HeroSection from "@/components/HeroSection";
import { KeyCommandsSection } from "@/components/KeyCommandsSection";
import { StackedBranchesSection } from "@/components/StackedBranchesSection";
import { CtaSection } from "@/components/CtaSection";

export default function Home() {
    return (
        <div className='flex md:min-h-screen flex-col bg-zinc-950 text-zinc-400'>
            <main className='flex-1'>
                <HeroSection />
                <div className='hidden md:block'>
                    <StackedBranchesSection />
                    <KeyCommandsSection />
                    <CtaSection />
                </div>
            </main>
        </div>
    );
}
