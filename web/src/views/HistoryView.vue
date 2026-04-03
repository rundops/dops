<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { fetchHistory } from "../lib/api";
import { parseTimeInput } from "../lib/timeparse";
import type { ExecutionRecord } from "../lib/types";

const router = useRouter();

const records = ref<ExecutionRecord[]>([]);
const loading = ref(true);
const search = ref("");

// --- Time range picker ---
const timeOpen = ref(false);
const timeInput = ref("");
const timeLabel = ref("Live · Past 15 Minutes");
const timeLive = ref(true);
const timeRange = ref<{ from: Date | null; to: Date | null }>({
  from: new Date(Date.now() - 15 * 60 * 1000),
  to: null,
});
const timeEl = ref<HTMLElement | null>(null);

const presets = [
  { shorthand: "15m", label: "Past 15 Minutes", ms: 15 * 60 * 1000 },
  { shorthand: "30m", label: "Past 30 Minutes", ms: 30 * 60 * 1000 },
  { shorthand: "1h", label: "Past 1 Hour", ms: 60 * 60 * 1000 },
  { shorthand: "4h", label: "Past 4 Hours", ms: 4 * 60 * 60 * 1000 },
  { shorthand: "1d", label: "Past 1 Day", ms: 24 * 60 * 60 * 1000 },
  { shorthand: "2d", label: "Past 2 Days", ms: 2 * 24 * 60 * 60 * 1000 },
  { shorthand: "1w", label: "Past 1 Week", ms: 7 * 24 * 60 * 60 * 1000 },
  { shorthand: "2w", label: "Past 2 Weeks", ms: 14 * 24 * 60 * 60 * 1000 },
];

function selectPreset(preset: { label: string; ms: number }) {
  timeRange.value = { from: new Date(Date.now() - preset.ms), to: null };
  timeLive.value = true;
  timeLabel.value = "Live · " + preset.label;
  timeOpen.value = false;
  timeInput.value = "";
}

function applyCustom(override?: string) {
  const input = override ?? timeInput.value;
  const parsed = parseTimeInput(input);
  if (parsed) {
    timeRange.value = { from: parsed.from, to: parsed.to };
    // Live if open-ended (to=null, from is relative)
    timeLive.value = parsed.to === null;
    timeLabel.value = timeLive.value ? "Live · " + parsed.label : parsed.label;
    timeOpen.value = false;
    timeInput.value = "";
  }
}

function clearTime() {
  timeRange.value = { from: new Date(Date.now() - 15 * 60 * 1000), to: null };
  timeLive.value = true;
  timeLabel.value = "Live · Past 15 Minutes";
  timeOpen.value = false;
  timeInput.value = "";
}

function onTimeKeydown(e: KeyboardEvent) {
  if (e.key === "Enter") applyCustom();
  if (e.key === "Escape") { timeOpen.value = false; timeInput.value = ""; }
}

function onClickOutside(e: MouseEvent) {
  if (timeEl.value && !timeEl.value.contains(e.target as Node)) timeOpen.value = false;
}

onMounted(async () => {
  records.value = await fetchHistory();
  loading.value = false;
  document.addEventListener("click", onClickOutside);
});
onUnmounted(() => { document.removeEventListener("click", onClickOutside); });

// --- Filtering ---
const filtered = computed(() => {
  let result = records.value;
  if (timeRange.value.from) {
    const from = timeRange.value.from.getTime();
    const to = timeRange.value.to ? timeRange.value.to.getTime() : Date.now();
    result = result.filter((r) => {
      const t = new Date(r.start_time).getTime();
      return t >= from && t <= to;
    });
  }
  if (search.value) {
    const q = search.value.toLowerCase();
    result = result.filter((r) =>
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
    case "success": return "bg-success-muted text-success";
    case "failed": return "bg-error-muted text-error";
    case "cancelled": return "bg-warning-muted text-warning";
    default: return "bg-primary-muted text-primary";
  }
}

function statusIcon(status: string): string {
  switch (status) {
    case "success": return "✓";
    case "failed": return "✕";
    case "cancelled": return "⊘";
    default: return "●";
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
    month: "short", day: "numeric",
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

        <!-- Time range trigger -->
        <div ref="timeEl" class="relative">
          <button
            @click.stop="timeOpen = !timeOpen"
            :class="timeLive
              ? 'border-success/40 bg-success-muted text-success hover:border-success'
              : 'border-border bg-bg text-fg-muted hover:border-border-active hover:text-fg'"
            class="flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium border rounded-md cursor-pointer transition-colors duration-150"
          >
            <svg v-if="timeLive" width="12" height="12" viewBox="0 0 16 16" fill="currentColor" class="shrink-0">
              <path d="M8 4a4 4 0 100 8 4 4 0 000-8z"/>
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 16 16" fill="currentColor" class="text-fg-subtle shrink-0">
              <path d="M1.5 8a6.5 6.5 0 1113 0 6.5 6.5 0 01-13 0zM8 0a8 8 0 100 16A8 8 0 008 0zm.5 4.75a.75.75 0 00-1.5 0v3.5a.75.75 0 00.37.65l2.5 1.5a.75.75 0 00.76-1.3L8.5 7.87V4.75z"/>
            </svg>
            <span class="truncate max-w-[160px]">{{ timeLabel }}</span>
            <button
              v-if="timeLabel !== 'All time'"
              @click.stop="clearTime"
              class="flex items-center text-fg-subtle hover:text-fg bg-transparent border-none cursor-pointer p-0 ml-0.5 text-[10px]"
              title="Clear"
            >✕</button>
            <svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor" class="text-fg-subtle">
              <path d="M4.427 7.427l3.396 3.396a.25.25 0 00.354 0l3.396-3.396A.25.25 0 0011.396 7H4.604a.25.25 0 00-.177.427z"/>
            </svg>
          </button>

          <!-- Two-panel dropdown -->
          <div
            v-if="timeOpen"
            class="absolute right-0 top-9 w-[520px] bg-bg-panel border border-border rounded-lg shadow-xl z-50 flex overflow-hidden"
          >
            <!-- Left panel: custom input + examples -->
            <div class="flex-1 p-4 border-r border-border">
              <div class="text-[12px] text-fg-muted mb-2.5 font-medium">Type custom times like:</div>
              <input
                v-model="timeInput"
                @keydown="onTimeKeydown"
                type="text"
                placeholder="e.g. 45m, last 2 hours, since 4/1..."
                class="w-full px-2.5 py-2 text-[13px] bg-bg border border-border rounded-md text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none mb-3"
                autofocus
              />

              <div class="text-[11px] text-fg-subtle mb-1.5 font-medium">Relative</div>
              <div class="flex flex-wrap gap-1.5 mb-3">
                <button
                  v-for="ex in ['45m', '12 hours', '10d', '2 weeks', 'last month', 'yesterday', 'today']"
                  :key="ex"
                  @click="applyCustom(ex)"
                  class="px-2 py-0.5 text-[11px] font-mono bg-bg-element border border-border/50 rounded text-fg-muted hover:text-fg hover:border-border-active cursor-pointer transition-colors duration-100"
                >{{ ex }}</button>
              </div>

              <div class="text-[11px] text-fg-subtle mb-1.5 font-medium">Fixed</div>
              <div class="flex flex-wrap gap-1.5 mb-3">
                <button
                  v-for="ex in ['Apr 1', 'Apr 1 - Apr 2', '4/1', '4/1 - 4/2']"
                  :key="ex"
                  @click="applyCustom(ex)"
                  class="px-2 py-0.5 text-[11px] font-mono bg-bg-element border border-border/50 rounded text-fg-muted hover:text-fg hover:border-border-active cursor-pointer transition-colors duration-100"
                >{{ ex }}</button>
              </div>

              <div class="text-[11px] text-fg-subtle mb-1.5 font-medium">Growing</div>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="ex in ['since 4/1', 'since yesterday']"
                  :key="ex"
                  @click="applyCustom(ex)"
                  class="px-2 py-0.5 text-[11px] font-mono bg-bg-element border border-border/50 rounded text-fg-muted hover:text-fg hover:border-border-active cursor-pointer transition-colors duration-100"
                >{{ ex }}</button>
              </div>
            </div>

            <!-- Right panel: presets -->
            <div class="w-[200px] py-1 overflow-y-auto">
              <button
                v-for="preset in presets"
                :key="preset.shorthand"
                @click="selectPreset(preset)"
                :class="timeLabel === preset.label
                  ? 'bg-primary-muted text-primary'
                  : 'text-fg-muted hover:bg-bg-hover hover:text-fg'"
                class="flex items-center gap-2.5 w-full text-left px-3 py-2 text-[12px] cursor-pointer transition-colors duration-100 border-none bg-transparent"
              >
                <span class="font-mono text-[10px] px-1.5 py-0.5 bg-bg-element rounded text-fg-subtle w-[32px] text-center shrink-0">
                  {{ preset.shorthand }}
                </span>
                <span>{{ preset.label }}</span>
              </button>
              <div class="border-t border-border mt-1 pt-1">
                <button
                  @click="clearTime"
                  :class="timeLabel === 'All time'
                    ? 'bg-primary-muted text-primary'
                    : 'text-fg-muted hover:bg-bg-hover hover:text-fg'"
                  class="flex items-center gap-2.5 w-full text-left px-3 py-2 text-[12px] cursor-pointer transition-colors duration-100 border-none bg-transparent"
                >
                  <span class="font-mono text-[10px] px-1.5 py-0.5 bg-bg-element rounded text-fg-subtle w-[32px] text-center shrink-0">∞</span>
                  <span>All Time</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Search -->
      <div class="relative">
        <svg class="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-subtle pointer-events-none" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
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
      <div v-if="loading" class="flex items-center justify-center h-32 text-fg-muted text-[13px]">Loading...</div>
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
              <span :class="statusClass(rec.status)" class="inline-flex items-center justify-center w-5 h-5 text-[10px] font-bold rounded-full shrink-0">{{ statusIcon(rec.status) }}</span>
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
