---
layout: home

hero:
  name: dops
  text: the do(ops) cli
  tagline: A browsable catalog of automation scripts that operators can select, parameterize, and execute directly from the terminal — or in the browser.
  actions:
    - theme: brand
      text: Try Live Demo
      link: https://demo.rundops.dev
    - theme: alt
      text: Get Started
      link: /guides/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/rundops/dops

features:
  - icon: "🖥️"
    title: Interactive TUI
    details: Full-screen terminal UI with sidebar navigation, parameter wizards, live streaming output, and risk confirmation gates.
  - icon: "🌐"
    title: Web UI
    details: Browser-based interface via dops open with real-time log streaming, parameter forms, and full theme support.
  - icon: "🤖"
    title: MCP Server
    details: Expose runbooks as tools for AI agents via the Model Context Protocol. Stdio and HTTP transports.
  - icon: "📦"
    title: Catalog System
    details: Organize runbooks locally or install shared catalogs from git repos. Aliases, risk policies, and encrypted vault.
  - icon: "⌨️"
    title: CLI Mode
    details: Run any runbook non-interactively with dops run. Pass parameters via flags for scripting and CI/CD.
  - icon: "🎨"
    title: 20 Themes
    details: github, dracula, gruvbox, nord, synthwave, catppuccin, and more. Custom themes via JSON.
---

<style>
.demo-section {
  max-width: 960px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.demo-section h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: var(--vp-c-text-1);
}
.demo-section p {
  color: var(--vp-c-text-2);
  margin-bottom: 1.25rem;
  font-size: 0.95rem;
}
.demo-section img {
  border-radius: 12px;
  border: 1px solid var(--vp-c-border);
  width: 100%;
}
.demo-divider {
  max-width: 960px;
  margin: 0 auto;
  border-top: 1px solid var(--vp-c-border);
}
.commands-section {
  max-width: 960px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.commands-section h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}
.commands-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 0.75rem;
}
.cmd-card {
  display: block;
  padding: 1rem 1.25rem;
  border: 1px solid var(--vp-c-border);
  border-radius: 10px;
  text-decoration: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.cmd-card:hover {
  border-color: var(--vp-c-brand-1);
  box-shadow: 0 0 0 1px var(--vp-c-brand-soft);
}
.cmd-card code {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--vp-c-brand-1);
}
.cmd-card span {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
}
.try-cta {
  max-width: 960px;
  margin: 0 auto;
  padding: 3rem 1.5rem;
  text-align: center;
}
.try-cta h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: var(--vp-c-text-1);
}
.try-cta p {
  color: var(--vp-c-text-2);
  margin-bottom: 1.5rem;
  font-size: 0.95rem;
}
.try-btn {
  display: inline-block;
  padding: 0.75rem 2rem;
  background: var(--vp-c-brand-1);
  color: var(--vp-c-bg) !important;
  font-weight: 600;
  font-size: 0.95rem;
  border-radius: 8px;
  text-decoration: none;
  transition: opacity 0.2s, transform 0.2s;
}
.try-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}
</style>

<div class="demo-section">
  <h2>Terminal UI</h2>
  <p>Navigate catalogs, fill parameters, confirm risk, and watch live output — all from the terminal.</p>
  <img src="https://raw.githubusercontent.com/rundops/dops/main/assets/demo.gif" alt="dops TUI demo" />
</div>

<div class="demo-divider"></div>

<div class="demo-section">
  <h2>Web UI</h2>
  <p>The same experience in the browser. Launch with <code>dops open</code>.</p>
  <img src="https://raw.githubusercontent.com/rundops/dops/main/assets/web-demo.gif" alt="dops web UI demo" />
</div>

<div class="demo-divider"></div>

<div class="demo-section">
  <h2>MCP Server</h2>
  <p>Expose runbooks as tools for AI agents. Run with <code>dops mcp serve</code>.</p>
  <img src="https://raw.githubusercontent.com/rundops/dops/main/assets/mcp-demo.gif" alt="dops MCP demo" />
</div>

<div class="demo-divider"></div>

<div class="try-cta">
  <h2>See it for yourself</h2>
  <p>No install needed. Browse runbooks, fill parameters, and run scripts in a live sandbox.</p>
  <a class="try-btn" href="https://demo.rundops.dev">Try the Live Demo &rarr;</a>
</div>

<div class="demo-divider"></div>

<div class="commands-section">
  <h2>Commands</h2>
  <div class="commands-grid">
    <a class="cmd-card" href="/reference/cli/dops">
      <code>dops</code>
      <span>Launch the interactive TUI</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-run">
      <code>dops run</code>
      <span>Execute a runbook non-interactively</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-open">
      <code>dops open</code>
      <span>Launch the web UI in a browser</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-init">
      <code>dops init</code>
      <span>Initialize configuration</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-catalog">
      <code>dops catalog</code>
      <span>Manage runbook catalogs</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-config">
      <code>dops config</code>
      <span>Read and write configuration</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-mcp">
      <code>dops mcp</code>
      <span>MCP server for AI agents</span>
    </a>
    <a class="cmd-card" href="/reference/cli/dops-completion">
      <code>dops completion</code>
      <span>Generate shell completions</span>
    </a>
  </div>
</div>
