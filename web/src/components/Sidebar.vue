<script setup lang="ts">
import { ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import type { Catalog } from "../lib/types";

defineProps<{
  catalogs: Catalog[];
  loading: boolean;
}>();

const router = useRouter();
const route = useRoute();
const collapsed = ref<Record<string, boolean>>({});
const filter = ref("");

function toggle(name: string) {
  collapsed.value[name] = !collapsed.value[name];
}

function filteredRunbooks(catalog: Catalog) {
  if (!filter.value) return catalog.runbooks;
  const q = filter.value.toLowerCase();
  return catalog.runbooks.filter(
    (rb) =>
      rb.name.toLowerCase().includes(q) ||
      rb.description.toLowerCase().includes(q)
  );
}

function isActive(rbId: string): boolean {
  return route.path === `/runbook/${rbId}`;
}

function riskDotClass(level: string): string {
  switch (level) {
    case "critical":
      return "bg-error";
    case "high":
      return "bg-warning";
    case "medium":
      return "bg-primary";
    default:
      return "bg-fg-subtle";
  }
}
</script>

<template>
  <aside
    class="w-[280px] min-w-[280px] border-r border-border bg-bg-panel flex flex-col h-full overflow-hidden"
  >
    <!-- Header -->
    <div class="px-5 pt-5 pb-4 border-b border-border flex items-center justify-between">
      <button
        @click="router.push('/')"
        class="text-[15px] font-bold text-fg flex items-center gap-2 bg-transparent border-none cursor-pointer p-0 hover:opacity-80 transition-opacity duration-150"
      >
        <span class="text-primary font-mono text-[13px] font-semibold">dops</span>
        runbooks
      </button>
      <div class="flex items-center gap-2.5">
        <a
          href="https://jacobhuemmer.github.io/dops-cli/"
          target="_blank"
          rel="noopener noreferrer"
          class="text-fg-subtle hover:text-fg-muted transition-colors duration-150"
          title="Documentation"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M0 1.75A.75.75 0 01.75 1h4.253c1.227 0 2.317.59 3 1.501A3.744 3.744 0 0111.006 1h4.245a.75.75 0 01.75.75v10.5a.75.75 0 01-.75.75h-4.507a2.25 2.25 0 00-1.591.659l-.622.621a.75.75 0 01-1.06 0l-.623-.621A2.25 2.25 0 005.258 13H.75a.75.75 0 01-.75-.75zm7.251 10.324l.004-5.073-.002-2.253A2.25 2.25 0 005.003 2.5H1.5v9h3.757a3.75 3.75 0 011.994.574zM8.755 4.75l-.004 7.322a3.752 3.752 0 011.992-.572H14.5v-9h-3.495a2.25 2.25 0 00-2.25 2.25z"/></svg>
        </a>
        <a
          href="https://github.com/jacobhuemmer/dops-cli"
          target="_blank"
          rel="noopener noreferrer"
          class="text-fg-subtle hover:text-fg-muted transition-colors duration-150"
          title="View on GitHub"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
        </a>
      </div>
    </div>

    <!-- Search -->
    <div class="px-4 py-3">
      <input
        v-model="filter"
        type="text"
        placeholder="Search runbooks\u2026"
        class="w-full px-3 py-[7px] text-[13px] bg-bg border border-border rounded-md text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none transition-colors duration-150"
      />
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto px-2 pt-1 pb-4">
      <div v-if="loading" class="text-fg-muted text-sm p-2">Loading...</div>

      <div v-for="cat in catalogs" :key="cat.name" class="mb-1">
        <!-- Catalog label -->
        <button
          @click="toggle(cat.name)"
          class="flex items-center gap-1.5 w-full text-left px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-muted hover:text-fg cursor-pointer select-none"
        >
          <span class="text-[9px] transition-transform duration-150" :class="collapsed[cat.name] ? '-rotate-90' : ''">&#9660;</span>
          {{ cat.display_name || cat.name }}
        </button>

        <!-- Runbook items -->
        <div v-if="!collapsed[cat.name]">
          <button
            v-for="rb in filteredRunbooks(cat)"
            :key="rb.id"
            @click="router.push(`/runbook/${rb.id}`)"
            :class="isActive(rb.id)
              ? 'bg-primary-muted text-primary'
              : 'text-fg-muted hover:bg-bg-hover hover:text-fg'"
            class="flex items-center gap-2 w-full text-left pl-6 pr-3 py-1.5 text-[13px] rounded-md cursor-pointer transition-all duration-100"
          >
            <span class="flex-1 truncate">{{ rb.name }}</span>
            <span
              v-if="rb.risk_level"
              :class="riskDotClass(rb.risk_level)"
              class="w-1.5 h-1.5 rounded-full shrink-0"
            ></span>
          </button>
        </div>
      </div>
    </nav>
  </aside>
</template>
