<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { fetchHistory } from "../lib/api";
import type { ExecutionRecord } from "../lib/types";

const router = useRouter();

const records = ref<ExecutionRecord[]>([]);
const loading = ref(true);
const search = ref("");

// --- Time range picker ---
const timePickerOpen = ref(false);
const timeQuery = ref("");
const timeLabel = ref("All time");
const timeRange = ref<{ from: Date | null; to: Date | null }>({ from: null, to: null });
const timePickerEl = ref<HTMLElement | null>(null);

const presets = [
  { label: "Last 2 minutes", ms: 2 * 60 * 1000 },
  { label: "Last 5 minutes", ms: 5 * 60 * 1000 },
  { label: "Last 15 minutes", ms: 15 * 60 * 1000 },
  { label: "Last 1 hour", ms: 60 * 60 * 1000 },
  { label: "Last 4 hours", ms: 4 * 60 * 60 * 1000 },
  { label: "Last 1 day", ms: 24 * 60 * 60 * 1000 },
  { label: "Last 7 days", ms: 7 * 24 * 60 * 60 * 1000 },
  { label: "Last 30 days", ms: 30 * 24 * 60 * 60 * 1000 },
  { label: "All time", ms: 0 },
];

const filteredPresets = computed(() => {
  if (!timeQuery.value) return presets;
  const q = timeQuery.value.toLowerCase();
  return presets.filter((p) => p.label.toLowerCase().includes(q));
});

// Parse custom date input: "2026-04-01 - 2026-04-03" or "2026-04-01"
function parseCustomRange(input: string): { from: Date; to: Date } | null {
  const parts = input.split(/\s*[-–]\s*/);
  if (parts.length === 2) {
    const from = new Date(parts[0].trim());
    const to = new Date(parts[1].trim());
    if (!isNaN(from.getTime()) && !isNaN(to.getTime())) {
      to.setHours(23, 59, 59, 999);
      return { from, to };
    }
  }
  if (parts.length === 1) {
    const d = new Date(parts[0].trim());
    if (!isNaN(d.getTime())) {
      const to = new Date(d);
      to.setHours(23, 59, 59, 999);
      return { from: d, to };
    }
  }
  return null;
}

function selectPreset(preset: { label: string; ms: number }) {
  timeLabel.value = preset.label;
  if (preset.ms === 0) {
    timeRange.value = { from: null, to: null };
  } else {
    timeRange.value = { from: new Date(Date.now() - preset.ms), to: null };
  }
  timePickerOpen.value = false;
  timeQuery.value = "";
}

function applyCustomRange() {
  const parsed = parseCustomRange(timeQuery.value);
  if (parsed) {
    timeRange.value = parsed;
    timeLabel.value = timeQuery.value;
    timePickerOpen.value = false;
    timeQuery.value = "";
  }
}

function onTimeKeydown(e: KeyboardEvent) {
  if (e.key === "Enter") {
    // Try custom range first, then first matching preset
    const parsed = parseCustomRange(timeQuery.value);
    if (parsed) {
      applyCustomRange();
    } else if (filteredPresets.value.length > 0) {
      selectPreset(filteredPresets.value[0]);
    }
  }
  if (e.key === "Escape") {
    timePickerOpen.value = false;
    timeQuery.value = "";
  }
}

function onClickOutside(e: MouseEvent) {
  if (timePickerEl.value && !timePickerEl.value.contains(e.target as Node)) {
    timePickerOpen.value = false;
  }
}

onMounted(async () => {
  records.value = await fetchHistory();
  loading.value = false;
  document.addEventListener("click", onClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", onClickOutside);
});

// --- Filtering ---
const filtered = computed(() => {
  let result = records.value;

  // Time range filter
  if (timeRange.value.from) {
    const from = timeRange.value.from.getTime();
    const to = timeRange.value.to ? timeRange.value.to.getTime() : Date.now();
    result = result.filter((r) => {
      const t = new Date(r.start_time).getTime();
      return t >= from && t <= to;
    });
  }

  // Text search
  if (search.value) {
    const q = search.value.toLowerCase();
    result = result.filter(
      (r) =>
        r.runbook_id.toLowerCase().includes(q) ||
        r.runbook_name.toLowerCase().includes(q) ||
        r.catalog_name.toLowerCase().includes(q) ||
        r.status.toLowerCase().includes(q) ||
        r.interface.toLowerCase().includes(q)
    );
  }

  return result;
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
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="px-5 py-3 border-b border-border bg-bg-panel">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-3">
          <span class="text-[15px] font-bold text-fg">History</span>
          <span class="text-fg-subtle text-[12px]" v-if="!loading">{{ filtered.length }} runs</span>
        </div>

        <!-- Time range picker -->
        <div ref="timePickerEl" class="relative">
          <button
            @click.stop="timePickerOpen = !timePickerOpen"
            class="flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium border border-border rounded-md bg-bg text-fg-muted hover:border-border-active hover:text-fg cursor-pointer transition-colors duration-150"
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" class="text-fg-subtle">
              <path d="M1.5 8a6.5 6.5 0 1113 0 6.5 6.5 0 01-13 0zM8 0a8 8 0 100 16A8 8 0 008 0zm.5 4.75a.75.75 0 00-1.5 0v3.5a.75.75 0 00.37.65l2.5 1.5a.75.75 0 00.76-1.3L8.5 7.87V4.75z"/>
            </svg>
            {{ timeLabel }}
            <svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor" class="text-fg-subtle ml-0.5">
              <path d="M4.427 7.427l3.396 3.396a.25.25 0 00.354 0l3.396-3.396A.25.25 0 0011.396 7H4.604a.25.25 0 00-.177.427z"/>
            </svg>
          </button>

          <!-- Dropdown -->
          <div
            v-if="timePickerOpen"
            class="absolute right-0 top-9 w-[260px] bg-bg-panel border border-border rounded-lg shadow-xl z-50 overflow-hidden"
          >
            <!-- Search input -->
            <div class="p-2 border-b border-border">
              <input
                v-model="timeQuery"
                @keydown="onTimeKeydown"
                type="text"
                placeholder="Type a time range..."
                class="w-full px-2.5 py-1.5 text-[12px] bg-bg border border-border rounded text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none"
                autofocus
              />
            </div>

            <!-- Presets -->
            <div class="max-h-[240px] overflow-y-auto py-1">
              <button
                v-for="preset in filteredPresets"
                :key="preset.label"
                @click="selectPreset(preset)"
                :class="timeLabel === preset.label
                  ? 'bg-primary-muted text-primary'
                  : 'text-fg-muted hover:bg-bg-hover hover:text-fg'"
                class="flex items-center w-full text-left px-3 py-1.5 text-[12px] cursor-pointer transition-colors duration-100 border-none bg-transparent"
              >
                <span class="w-4 text-center mr-2" v-if="timeLabel === preset.label">✓</span>
                <span class="w-4 mr-2" v-else></span>
                {{ preset.label }}
              </button>
            </div>

            <!-- Custom hint -->
            <div v-if="timeQuery && !filteredPresets.length" class="px-3 py-2 text-[11px] text-fg-subtle border-t border-border">
              Try: <span class="font-mono">2026-04-01 - 2026-04-03</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Search -->
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
        <span v-if="search || timeRange.from">No results matching your filters</span>
        <span v-else>No executions yet. Run a runbook to see history here.</span>
      </div>

      <div v-else class="divide-y divide-border/40">
        <div
          v-for="rec in filtered"
          :key="rec.id"
          @click="router.push(`/history/${rec.id}`)"
          class="px-5 py-3 hover:bg-bg-hover transition-colors duration-100 cursor-pointer group"
        >
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

          <div class="flex items-center gap-2 ml-[30px] text-[11px]">
            <span class="font-mono text-fg-muted">{{ rec.duration || "–" }}</span>
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
