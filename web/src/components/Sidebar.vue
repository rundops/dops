<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import type { Catalog } from "../lib/types";

defineProps<{
  catalogs: Catalog[];
  loading: boolean;
}>();

const router = useRouter();
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

function riskColor(level: string): string {
  switch (level) {
    case "critical":
      return "text-error";
    case "high":
      return "text-warning";
    case "medium":
      return "text-primary";
    default:
      return "text-fg-muted";
  }
}
</script>

<template>
  <aside
    class="w-64 border-r border-border bg-bg-panel flex flex-col h-full overflow-hidden"
  >
    <div class="p-3 border-b border-border">
      <h1 class="text-primary font-bold text-lg">dops</h1>
    </div>

    <div class="p-2">
      <input
        v-model="filter"
        type="text"
        placeholder="Filter..."
        class="w-full px-2 py-1 text-sm bg-bg-element border border-border rounded text-fg placeholder-fg-muted focus:border-border-active focus:outline-none"
      />
    </div>

    <nav class="flex-1 overflow-y-auto px-2 pb-2">
      <div v-if="loading" class="text-fg-muted text-sm p-2">Loading...</div>

      <div v-for="cat in catalogs" :key="cat.name" class="mb-1">
        <button
          @click="toggle(cat.name)"
          class="flex items-center gap-1 w-full text-left px-2 py-1 text-sm text-fg-muted hover:text-fg rounded"
        >
          <span class="text-xs">{{ collapsed[cat.name] ? "▶" : "▼" }}</span>
          <span>{{ cat.display_name || cat.name }}/</span>
        </button>

        <div v-if="!collapsed[cat.name]" class="ml-3">
          <button
            v-for="rb in filteredRunbooks(cat)"
            :key="rb.id"
            @click="router.push(`/runbook/${rb.id}`)"
            class="flex items-center gap-2 w-full text-left px-2 py-1 text-sm hover:bg-bg-element rounded group"
          >
            <span class="text-fg-muted group-hover:text-fg">├──</span>
            <span class="text-fg truncate">{{ rb.name }}</span>
            <span v-if="rb.risk_level" :class="riskColor(rb.risk_level)" class="text-xs ml-auto">
              {{ rb.risk_level }}
            </span>
          </button>
        </div>
      </div>
    </nav>
  </aside>
</template>
