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
    <div class="px-5 pt-5 pb-4 border-b border-border">
      <h1 class="text-[15px] font-bold text-fg flex items-center gap-2">
        <span class="text-primary font-mono text-[13px] font-semibold">dops</span>
        runbooks
      </h1>
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
