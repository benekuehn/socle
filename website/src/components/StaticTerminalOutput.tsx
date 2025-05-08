'use client'

import React from 'react';

export const StaticTerminalOutput: React.FC = () => {
  return (
    <>
      <div className="flex">
        <span className="text-zinc-500 mr-2">$</span>
        <span className="text-zinc-300">so log</span>
      </div>
      <div className="mt-2">
        <span className="text-green-500">●</span> main
        <br />
        <span className="ml-4 text-green-500">●</span> feature/auth
        <br />
        <span className="ml-8 text-green-500">●</span> feature/login-form <span className="text-zinc-400">(current)</span>
        <br />
        <span className="ml-12 text-yellow-500">○</span> feature/validation <span className="text-yellow-500">(needs rebase)</span>
      </div>
    </>
  );
}; 