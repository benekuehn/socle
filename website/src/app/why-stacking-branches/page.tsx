import { CtaSection } from "@/components/CtaSection";

export default function WhyStackingBranches() {
  return (
    <div className="flex min-h-screen flex-col bg-zinc-950 text-zinc-400">
      <main className="container mx-auto px-16 md:py-16 max-w-3xl">
        <header className="mb-16">
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-8 text-zinc-100 leading-tight">
            Unlock Effortless Flow: Understanding Stacked Branches and Diffs in Your Git Workflow
          </h1>
        </header>

        <div className="prose prose-invert max-w-none prose-headings:font-bold prose-headings:tracking-tight prose-p:leading-relaxed prose-p:text-zinc-300 prose-strong:text-zinc-100 prose-a:text-zinc-100 prose-a:no-underline hover:prose-a:underline">
          <section id="intro" className="mb-16">
            <p className="text-lg mb-8 leading-relaxed">
              In the journey of software development, maintaining momentum while ensuring code quality is paramount. Large, monolithic pull requests can often hinder this, leading to lengthy review cycles and a disrupted development rhythm. Discover a more considered approach: stacked branches, also known as stacked diffs or stacked changes. This purposeful methodology empowers you to break down complex features into a series of manageable, reviewable units, fostering a state of continuous, productive flow. It's an approach refined and extensively used by companies like Google and Meta to manage complexity and accelerate development at scale.
            </p>
            <p className="mb-6 text-lg">This page will guide you through understanding:</p>
            <ul className="space-y-3 text-lg">
              <li>What stacked branches are and why they matter.</li>
              <li>The benefits of integrating stacked diffs with Git and GitHub.</li>
              <li>How this approach compares to traditional workflows.</li>
              <li>How tools can support your journey into stacking.</li>
            </ul>
          </section>

          <section id="what-are-stacked-branches" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">What Are Stacked Branches (or Stacked Diffs)?</h2>
            <p className="mb-6 text-lg leading-relaxed">
              Imagine building a complex feature. Instead of one massive branch with dozens of commits, stacked branches involve creating a sequence of smaller, dependent branches. Each branch in the stack builds upon the previous one, containing a distinct, logical piece of the larger feature.
            </p>
            <p className="text-lg leading-relaxed">
              Think of it as writing a book chapter by chapter, rather than all at once. Each "chapter" (branch) can be reviewed and refined independently, yet contributes to the cohesive whole. With stacked diffs in Git, each branch represents a focused set of changes, making the review process more insightful and efficient.
            </p>
          </section>

          <section id="benefits" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">The Quiet Confidence of Stacking: Key Benefits</h2>
            <p className="mb-8 text-lg leading-relaxed">Adopting a stacked changes workflow isn't just about a new Git strategy; it's about enabling a more thoughtful and productive development experience.</p>
            <ul className="space-y-6">
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Focused Pull Requests, Better Reviews</strong>
                <span className="text-lg leading-relaxed">Smaller, self-contained PRs are significantly easier and faster for your team to review. This precision leads to more thorough feedback and higher code quality.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Maintained Momentum</strong>
                <span className="text-lg leading-relaxed">No more waiting for a massive PR to be approved. While one part of your stack is under review, you can continue building the next dependent feature on a new branch, keeping you in a state of productive flow.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Simplified Rebasing and Conflict Resolution</strong>
                <span className="text-lg leading-relaxed">Addressing changes or resolving conflicts becomes more manageable when dealing with smaller, incremental diffs rather than a large, complex branch.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Clearer Project Evolution</strong>
                <span className="text-lg leading-relaxed">The history of your project becomes a more understandable narrative of incremental progress, as each stacked PR tells a focused part of the story.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Early Integration & Feedback</strong>
                <span className="text-lg leading-relaxed">Integrate and test smaller pieces of functionality sooner, catching potential issues earlier in the development cycle.</span>
              </li>
            </ul>
          </section>

          <section id="stacking-with-git-github" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">Stacked Diffs with Git and GitHub: An Effortless Flow</h2>
            <p className="mb-8 text-lg leading-relaxed">The stacked diffs workflow integrates naturally with core Git concepts and is powerfully supported by platforms like GitHub.</p>
            <ol className="space-y-6 list-decimal pl-6">
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Base Branch</strong>
                <span className="text-lg leading-relaxed">You start with your main development branch (e.g., <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">main</code> or <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">develop</code>).</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">First Stacked Branch</strong>
                <span className="text-lg leading-relaxed">Create your first feature branch (e.g., <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">feature-a-part1</code>) from the base. Make your focused commits.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Second Stacked Branch</strong>
                <span className="text-lg leading-relaxed">Create your second branch (e.g., <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">feature-a-part2</code>) <em>from</em> <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">feature-a-part1</code>. Add the next logical set of changes.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Continue Stacking</strong>
                <span className="text-lg leading-relaxed">Repeat this process, creating a "stack" of branches, each dependent on the last.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Pull Requests on GitHub</strong>
                <span className="text-lg leading-relaxed">For each branch in your stack, you'll open a separate Pull Request on GitHub. It's helpful to indicate the dependency or order in your PR descriptions (e.g., "Part 1 of Feature X," "Builds on #123").</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Review and Merge</strong>
                <span className="text-lg leading-relaxed">Reviewers can tackle each PR individually. As a lower PR in the stack is approved and merged, you rebase the subsequent branches in your stack onto the updated base. For instance, if <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">feature-a-part1</code> is merged into <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">main</code>, you would rebase <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">feature-a-part2</code> (and any further branches) onto the new <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">main</code>.</span>
              </li>
            </ol>
            <p className="mt-8 text-lg leading-relaxed">While the core Git commands (<code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">branch</code>, <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">checkout</code>, <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">commit</code>, <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">rebase</code>) are the foundation, managing a tall stack can introduce some manual overhead. This is where purpose-built tools can offer significant assistance.</p>
          </section>

          <section id="stacked-vs-traditional" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">Stacked Changes vs. Long-Lived Feature Branches</h2>
            <p className="mb-8 text-lg leading-relaxed">Traditional workflows often involve long-lived feature branches where all changes for a feature accumulate before a single, large PR is opened.</p>
            <div className="overflow-x-auto rounded-lg border border-zinc-800">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="border-b border-zinc-800">
                    <th className="text-left py-4 px-6 text-zinc-200 font-semibold">Feature</th>
                    <th className="text-left py-4 px-6 text-zinc-200 font-semibold">Stacked Diffs Workflow</th>
                    <th className="text-left py-4 px-6 text-zinc-200 font-semibold">Traditional Feature Branch Workflow</th>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Pull Request Size</strong></td>
                    <td className="py-4 px-6">Small, focused</td>
                    <td className="py-4 px-6">Large, often complex</td>
                  </tr>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Review Cycle</strong></td>
                    <td className="py-4 px-6">Faster, more iterative</td>
                    <td className="py-4 px-6">Slower, can be overwhelming</td>
                  </tr>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Review Quality</strong></td>
                    <td className="py-4 px-6">Helpful comments</td>
                    <td className="py-4 px-6">LGTM</td>
                  </tr>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Developer Flow</strong></td>
                    <td className="py-4 px-6">Continuous, less blocking</td>
                    <td className="py-4 px-6">Can lead to bottlenecks and waiting</td>
                  </tr>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Integration Risk</strong></td>
                    <td className="py-4 px-6">Lower, issues caught early</td>
                    <td className="py-4 px-6">Higher, integration issues surface late</td>
                  </tr>
                  <tr className="border-b border-zinc-800">
                    <td className="py-4 px-6"><strong className="text-zinc-100">Code Merging</strong></td>
                    <td className="py-4 px-6">Incremental, manageable</td>
                    <td className="py-4 px-6">"Big bang" merge, potentially difficult</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p className="mt-8 text-lg leading-relaxed">The stacked approach, by contrast, is designed for clarity and forward momentum, enabling teams to make considered progress without the friction of unwieldy reviews.</p>
          </section>

          <section id="tools" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">Enabling Productive Flow: Purpose-Built Tools for Stacked Diffs</h2>
            <p className="mb-8 text-lg leading-relaxed">While the foundational principles of stacked diffs can be applied with core Git commands, purpose-built tools can make this powerful workflow feel truly effortless and intuitive. They handle the operational complexities, allowing you to focus on what you do best: crafting quality code.</p>

            <h3 className="text-2xl font-semibold mb-6 text-zinc-100 tracking-tight"><span className="text-zinc-100">Socle:</span> Simple, Elegant Stacking – No Fuss.</h3>
            <p className="mb-8 text-lg leading-relaxed"><span className="text-zinc-100">Socle</span> is designed with a singular focus: to make the sophisticated workflow of stacked diffs remarkably simple and seamlessly integrated into your daily development. We believe powerful tools don't need to be complicated.</p>
            <ul className="space-y-6 mb-12">
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Truly Open & Adaptable</strong>
                <span className="text-lg leading-relaxed">As a fully open-source solution under the permissive MIT license, <span className="text-zinc-100">Socle</span> empowers individuals and teams everywhere. It's free to use, modify, and enhance, ensuring it can be tailored to your specific needs.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Effortless by Design</strong>
                <span className="text-lg leading-relaxed"><span className="text-zinc-100">Socle</span> offers an elegant solution with simple, intuitive commands. It's built to get out of your way, enabling a natural, productive flow without a steep learning curve.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Powerful Yet Uncomplicated</strong>
                <span className="text-lg leading-relaxed">Experience the full benefits of stacked changes—focused PRs, smoother reviews, continuous progress—through a tool that prioritizes clarity and ease of use.</span>
              </li>
            </ul>

            <h3 className="text-2xl font-semibold mb-6 text-zinc-100 tracking-tight">The Broader Toolkit Landscape</h3>
            <p className="mb-8 text-lg leading-relaxed">While <span className="text-zinc-100">Socle</span> provides a uniquely accessible and powerful open-source path to stacking, the ecosystem includes other tools:</p>
            <ul className="space-y-6 mb-8">
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Command-Line Interfaces (CLIs)</strong>
                <span className="text-lg leading-relaxed">Various open-source CLIs offer different approaches to managing stacks, often with GitHub integration. These include <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">ghstack</code>, <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">git-town</code>, <code className="bg-zinc-800/50 px-2 py-1 rounded text-sm font-mono">spr</code>, and Aviator (which also offers an open-source CLI for managing stacked branches). Commercial options like Graphite also exist in this space.</span>
              </li>
            </ul>
          </section>

          <section id="getting-started" className="mb-16">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">Getting Started with Stacked Branches</h2>
            <p className="mb-8 text-lg leading-relaxed">Transitioning to a stacked workflow can be a gradual process:</p>
            <ol className="space-y-6 list-decimal pl-6">
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Understand the "Why"</strong>
                <span className="text-lg leading-relaxed">Ensure you and your team appreciate the benefits of smaller, focused PRs and iterative development.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Start Small</strong>
                <span className="text-lg leading-relaxed">Try stacking with just two or three small, related changes for a single feature.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Communicate</strong>
                <span className="text-lg leading-relaxed">Clearly label your PRs and explain the dependencies to your reviewers.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Explore Tooling</strong>
                <span className="text-lg leading-relaxed">As you become more comfortable, investigate tools like <span className="text-zinc-100">Socle</span> or consider how your existing setup can support this.</span>
              </li>
              <li className="flex flex-col">
                <strong className="text-xl text-zinc-100 mb-2">Iterate and Refine</strong>
                <span className="text-lg leading-relaxed">Like any process, adapt it to what works best for your team.</span>
              </li>
            </ol>
          </section>

          <section id="conclusion">
            <h2 className="text-3xl font-bold mb-8 text-zinc-100 tracking-tight">Embrace the Flow of Stacked Progress</h2>
            <p className="text-lg leading-relaxed">Stacked branches and diffs offer a considered, enabling approach to modern software development. By fostering focused pull requests, enabling better reviews, and keeping you in a state of productive flow, this workflow helps your team build complex features with greater clarity and confidence. It's about making progress thoughtfully, one well-crafted layer at a time.</p>
          </section>
        </div>
      </main>
      <CtaSection />
    </div>

  );
} 