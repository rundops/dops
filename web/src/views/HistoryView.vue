<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import { fetchHistory } from "../lib/api";
import type { ExecutionRecord } from "../lib/types";

const router = useRouter();

const records = ref<ExecutionRecord[]>([]);
const loading = ref(true);
const search = ref("");

onMounted(async () => {
  records.value = await fetchHistory();
  loading.value = false;
});

const filtered = computed(() => {
  if (!search.value) return records.value;
  const q = search.value.toLowerCase();
  return records.value.filter(
    (r) =>
      r.runbook_id.toLowerCase().includes(q) ||
      r.runbook_name.toLowerCase().includes(q) ||
      r.catalog_name.toLowerCase().includes(q) ||
      r.status.toLowerCase().includes(q) ||
      r.interface.toLowerCase().includes(q)
  );
});

function statusClass(status: string): string {
  switch (status) {
    case "success":
      return "bg-success-muted text-success";
    case "failed":
      return "bg-error-muted text-error";
    case "cancelled":
      return "bg-warning-muted text-warning";
    default:
      return "bg-primary-muted text-primary";
  }
}

function statusIcon(status: string): string {
  switch (status) {
    case "success":
      return "✓";
    case "failed":
      return "✕";
    case "cancelled":
      return "⊘";
    default:
      return "●";
  }
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  const mins = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;

  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: now.getFullYear() !== d.getFullYear() ? "numeric" : undefined,
  });
}

function formatDuration(dur: string | undefined): string {
  if (!dur) return "–";
  // Strip trailing "ms" or "s" noise for cleaner display
  return dur;
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="px-5 py-3 border-b border-border bg-bg-panel">
      <div class="flex items-center gap-3 mb-3">
        <span class="text-[15px] font-bold text-fg">History</span>
        <span class="text-fg-subtle text-[12px]" v-if="!loading">{{ filtered.length }} runs</span>
      </div>
      <div class="relative">
        <svg
          class="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-subtle pointer-events-none"
          width="14" height="14" viewBox="0 0 16 16" fill="currentColor"
        >
          <path fill-rule="evenodd" d="M11.5 7a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zm-.82 4.74a6 6 0 111.06-1.06l3.04 3.04a.75.75 0 11-1.06 1.06l-3.04-3.04z"/>
        </svg>
        <input
          v-model="search"
          type="text"
          placeholder="Search by runbook, status, interface..."
          class="w-full pl-8 pr-3 py-[7px] text-[13px] bg-bg border border-border rounded-md text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none transition-colors duration-150"
        />
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto">
      <div v-if="loading" class="flex items-center justify-center h-32 text-fg-muted text-[13px]">
        Loading...
      </div>

      <div v-else-if="filtered.length === 0" class="flex flex-col items-center justify-center h-32 text-fg-subtle text-[13px]">
        <span v-if="search">No results for "{{ search }}"</span>
        <span v-else>No executions yet. Run a runbook to see history here.</span>
      </div>

      <div v-else class="divide-y divide-border/40">
        <div
          v-for="rec in filtered"
          :key="rec.id"
          @click="router.push(`/history/${rec.id}`)"
          class="px-5 py-3 hover:bg-bg-hover transition-colors duration-100 cursor-pointer group"
        >
          <!-- Row 1: Runbook + Status + Time -->
          <div class="flex items-center justify-between mb-1">
            <div class="flex items-center gap-2.5 min-w-0">
              <span
                :class="statusClass(rec.status)"
                class="inline-flex items-center justify-center w-5 h-5 text-[10px] font-bold rounded-full shrink-0"
              >
                {{ statusIcon(rec.status) }}
              </span>
              <span class="font-mono text-[13px] text-fg truncate">{{ rec.runbook_id }}</span>
            </div>
            <span class="text-[11px] text-fg-subtle whitespace-nowrap ml-3">{{ formatTime(rec.start_time) }}</span>
          </div>

          <!-- Row 2: Metadata pills -->
          <div class="flex items-center gap-2 ml-[30px] text-[11px]">
            <span class="font-mono text-fg-muted">{{ formatDuration(rec.duration) }}</span>
            <span class="text-fg-subtle">·</span>
            <span class="font-mono px-1.5 py-0.5 bg-bg-element rounded text-fg-subtle">{{ rec.interface.toUpperCase() }}</span>
            <span class="text-fg-subtle">·</span>
            <span class="text-fg-muted">{{ rec.output_lines }} lines</span>
            <span v-if="rec.exit_code !== 0" class="text-fg-subtle">·</span>
            <span v-if="rec.exit_code !== 0" class="text-error font-mono">exit {{ rec.exit_code }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
