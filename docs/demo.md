---
layout: page
title: Demo
---

<style>
.demo-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1.5rem 1rem;
}
.demo-header {
  text-align: center;
  margin-bottom: 1.5rem;
}
.demo-header h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: var(--vp-c-text-1);
}
.demo-header p {
  color: var(--vp-c-text-2);
  font-size: 0.95rem;
  max-width: 600px;
  margin: 0 auto;
}
.demo-frame {
  width: 100%;
  height: 80vh;
  border: 1px solid var(--vp-c-border);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
}
.demo-hint {
  display: flex;
  justify-content: center;
  gap: 2rem;
  margin-top: 1.25rem;
  flex-wrap: wrap;
}
.demo-hint span {
  color: var(--vp-c-text-3);
  font-size: 0.85rem;
}
.demo-hint code {
  font-size: 0.8rem;
  color: var(--vp-c-brand-1);
}
</style>

<div class="demo-container">
  <div class="demo-header">
    <h1>Try dops</h1>
    <p>A live sandbox with sample runbooks. Browse catalogs, fill parameters, and execute scripts — no install required.</p>
  </div>
  <iframe class="demo-frame" src="https://demo.rundops.dev" allow="clipboard-write" loading="lazy"></iframe>
  <div class="demo-hint">
    <span>Install locally: <code>brew tap rundops/tap && brew install dops && dops open</code></span>
  </div>
</div>
