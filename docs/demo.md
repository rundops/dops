---
layout: page
title: Demo
---

<style>
.demo-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem;
}
.demo-container h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: var(--vp-c-text-1);
}
.demo-container p {
  color: var(--vp-c-text-2);
  margin-bottom: 1.25rem;
  font-size: 0.95rem;
}
.demo-frame {
  width: 100%;
  height: 80vh;
  border: 1px solid var(--vp-c-border);
  border-radius: 12px;
  background: var(--vp-c-bg-alt);
}
</style>

<div class="demo-container">
  <h1>Try dops</h1>
  <p>Browse runbooks, fill parameters, and run scripts — all in the browser. This is a live sandbox with sample runbooks.</p>
  <iframe class="demo-frame" src="https://demo.rundops.dev" allow="clipboard-write" loading="lazy"></iframe>
</div>
