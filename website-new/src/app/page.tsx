import HeroSection from "@/components/HeroSection";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-zinc-950 text-zinc-400">
      <main className="flex-1">
        <HeroSection />
      </main>
    </div>
  );
}
