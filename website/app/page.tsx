"use client"

import { useState } from "react"
import Header from "../components/landing/Header"
import HeroSection from "../components/landing/HeroSection"
import KeyCommandsSection from "../components/landing/KeyCommandsSection"
import CtaSection from "../components/landing/CtaSection"
import Footer from "../components/landing/Footer"
import { TerminalDemo } from "../components/TerminalDemo"

export default function Home() {


  return (
    <div className="flex min-h-screen flex-col bg-black text-white">
      {/* <Header /> */}
      <main className="flex-1">
        <HeroSection />
        {/* <TerminalDemo />
        <KeyCommandsSection />
        <CtaSection copied={copied} handleCopy={handleCopy} /> */}
      </main>
      {/* <Footer /> */}
    </div>
  )
}
