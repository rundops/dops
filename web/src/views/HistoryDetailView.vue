<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import { fetchHistoryRecord, fetchHistoryLog } from "../lib/api";
import { AnsiUp } from "ansi_up";
import type { ExecutionRecord } from "../lib/types";

const props = defineProps<{ id: string }>();
const router = useRouter();

const record = ref<ExecutionRecord | null>(null);
const logLines = ref<string[]>([]);
const logAvailable = ref(false);
const loading = ref(true);

const ansi = new AnsiUp();
ansi.use_classes = false;

function renderAnsi(text: string): string {
  return ansi.ansi_to_html(text);
}

function statusPillClass(status: string): string {
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

const statusText = computed(() => {
  if (!record.value) return "";
  const label = record.value.status.charAt(0).toUpperCase() + record.value.status.slice(1);
  return record.value.duration ? `${label} · ${record.value.duration}` : label;
});

onMounted(async () => {
  try {
    record.value = await fetchHistoryRecord(props.id);
    const log = await fetchHistoryLog(props.id);
    logLines.value = log.lines;
    logAvailable.value = log.available;
  } catch {
    // Record not found
  }
  loading.value = false;
});
</script>

<template>
  <div class="flex flex-col h-full">
    <div v-if="loading" class="p-6 text-fg-muted text-[13px]">Loading...</div>
    <div v-else-if="!record" class="p-6 text-fg-muted text-[13px]">Execution record not found.</div>
    <template v-else>
      <!-- Header -->
      <div class="px-5 py-3 border-b border-border flex items-center justify-between bg-bg-panel">
        <div class="flex items-center gap-3 min-w-0">
          <button
            @click="router.push('/history')"
            class="flex items-center text-fg-subtle hover:text-fg transition-colors duration-150 bg-transparent border-none cursor-pointer p-0 shrink-0"
            title="Back to history"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path fill-rule="evenodd" d="M7.78 12.53a.75.75 0 01-1.06 0L2.47 8.28a.75.75 0 010-1.06l4.25-4.25a.75.75 0 011.06 1.06L4.81 7h7.44a.75.75 0 010 1.5H4.81l2.97 2.97a.75.75 0 010 1.06z"/></svg>
          </button>
          <span class="font-mono text-[13px] text-fg-muted px-2 py-0.5 bg-bg-element rounded truncate">{{ record.runbook_id }}</span>
          <span
            :class="statusPillClass(record.status)"
            class="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-full whitespace-nowrap"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-current shrink-0"></span>
            {{ statusText }}
          </span>
          <span class="text-fg-subtle text-[11px] font-mono">{{ logLines.length }} lines</span>
        </div>
        <span class="text-[10px] font-mono px-1.5 py-0.5 bg-bg-element rounded text-fg-subtle shrink-0">
          {{ record.interface.toUpperCase() }}
        </span>
      </div>

      <!-- Metadata bar -->
      <div class="px-5 py-2 border-b border-border/50 bg-bg-panel flex flex-wrap gap-x-6 gap-y-1 text-[12px] text-fg-muted">
        <span>
          <span class="text-fg-subtle">Time:</span>
          {{ new Date(record.start_time).toLocaleString() }}
        </span>
        <span v-if="record.parameters && Object.keys(record.parameters).length">
          <span class="text-fg-subtle">Params:</span>
          <span v-for="(v, k) in record.parameters" :key="k" class="ml-1 font-mono">{{ k }}={{ v }}</span>
        </span>
        <span v-if="record.exit_code !== 0">
          <span class="text-fg-subtle">Exit:</span> {{ record.exit_code }}
        </span>
      </div>

      <!-- Log output -->
      <div class="flex-1 overflow-y-auto px-6 py-4 bg-bg font-mono text-[13px] leading-[1.7]">
        <template v-if="logAvailable && logLines.length > 0">
          <div
            v-for="(line, i) in logLines"
            :key="i"
            class="flex gap-3 whitespace-pre-wrap break-all"
          >
            <span class="text-fg-subtle select-none text-right min-w-[28px] shrink-0">{{ i + 1 }}</span>
            <span class="text-fg-muted" v-html="renderAnsi(line)"></span>
          </div>
        </template>
        <div v-else-if="!logAvailable" class="text-fg-subtle">
          Log file no longer available.
          <span v-if="record.output_summary" class="block mt-2 text-fg-muted">
            Last output: {{ record.output_summary }}
          </span>
        </div>
        <div v-else class="text-fg-subtle">No output recorded.</div>
      </div>
    </template>
  </div>
</template>
