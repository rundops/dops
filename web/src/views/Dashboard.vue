<script setup lang="ts">
import { ref, onMounted } from "vue";
import { fetchCatalogs } from "../lib/api";

const catalogCount = ref(0);
const runbookCount = ref(0);
const highRiskCount = ref(0);

onMounted(async () => {
  try {
    const catalogs = await fetchCatalogs();
    catalogCount.value = catalogs.length;
    runbookCount.value = catalogs.reduce((sum, c) => sum + c.runbooks.length, 0);
    highRiskCount.value = catalogs.reduce(
      (sum, c) =>
        sum +
        c.runbooks.filter(
          (rb) => rb.risk_level === "high" || rb.risk_level === "critical"
        ).length,
      0
    );
  } catch {
    // Sidebar already loads catalogs; dashboard stats are non-critical.
  }
});
</script>

<template>
  <div class="flex items-center justify-center h-full overflow-y-auto">
    <div class="text-center max-w-[480px] px-10">
      <!-- Logo -->
      <div class="font-mono text-4xl font-extrabold text-primary tracking-tight mb-2">
        dops
      </div>

      <!-- Tagline -->
      <p class="text-[15px] text-fg-muted mb-9">
        Select a runbook from the sidebar to get started.
      </p>

      <!-- Stats bar -->
      <div
        v-if="runbookCount"
        class="flex gap-px bg-border rounded-lg overflow-hidden mb-9"
      >
        <div class="flex-1 bg-bg-panel py-4 px-5 text-center">
          <div class="text-2xl font-bold font-mono text-fg">{{ catalogCount }}</div>
          <div class="text-xs text-fg-muted mt-1">Catalogs</div>
        </div>
        <div class="flex-1 bg-bg-panel py-4 px-5 text-center">
          <div class="text-2xl font-bold font-mono text-fg">{{ runbookCount }}</div>
          <div class="text-xs text-fg-muted mt-1">Runbooks</div>
        </div>
        <div class="flex-1 bg-bg-panel py-4 px-5 text-center">
          <div class="text-2xl font-bold font-mono text-fg">{{ highRiskCount }}</div>
          <div class="text-xs text-fg-muted mt-1">High Risk</div>
        </div>
      </div>

      <!-- Keyboard hint -->
      <div class="text-[13px] text-fg-subtle flex items-center justify-center gap-1.5">
        <kbd class="font-mono text-[11px] px-1.5 py-0.5 bg-bg-element border border-border rounded text-fg-muted">/</kbd>
        to search
        <span class="mx-1">&middot;</span>
        <kbd class="font-mono text-[11px] px-1.5 py-0.5 bg-bg-element border border-border rounded text-fg-muted">&uarr;&darr;</kbd>
        to navigate
      </div>
    </div>
  </div>
</template>
