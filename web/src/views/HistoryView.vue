<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { fetchHistory } from "../lib/api";
import type { ExecutionRecord } from "../lib/types";

const records = ref<ExecutionRecord[]>([]);
const loading = ref(true);
const filterStatus = ref("");
const filterRunbook = ref("");

async function load() {
  loading.value = true;
  records.value = await fetchHistory({
    runbook: filterRunbook.value || undefined,
    status: filterStatus.value || undefined,
  });
  loading.value = false;
}

onMounted(load);

const filtered = computed(() => records.value);

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

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function interfaceBadge(iface: string): string {
  return iface.toUpperCase();
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="px-5 py-3 border-b border-border bg-bg-panel flex items-center justify-between">
      <div class="flex items-center gap-3">
        <span class="text-[15px] font-bold text-fg">History</span>
        <span class="text-fg-muted text-[13px]" v-if="!loading">{{ filtered.length }} executions</span>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="filterStatus"
          @change="load()"
          class="px-2 py-1 text-[12px] bg-bg border border-border rounded text-fg-muted cursor-pointer"
        >
          <option value="">All statuses</option>
          <option value="success">Success</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
        <input
          v-model="filterRunbook"
          @keyup.enter="load()"
          type="text"
          placeholder="Filter by runbook..."
          class="px-2 py-1 text-[12px] bg-bg border border-border rounded text-fg placeholder-fg-subtle w-[180px]"
        />
      </div>
    </div>

    <!-- Table -->
    <div class="flex-1 overflow-y-auto">
      <div v-if="loading" class="p-6 text-fg-muted text-[13px]">Loading...</div>
      <div v-else-if="filtered.length === 0" class="p-6 text-fg-muted text-[13px]">No executions found.</div>
      <table v-else class="w-full text-[13px]">
        <thead class="sticky top-0 bg-bg-panel border-b border-border">
          <tr class="text-left text-fg-muted text-[11px] uppercase tracking-wide">
            <th class="px-5 py-2.5 font-semibold">Time</th>
            <th class="px-3 py-2.5 font-semibold">Runbook</th>
            <th class="px-3 py-2.5 font-semibold">Status</th>
            <th class="px-3 py-2.5 font-semibold">Duration</th>
            <th class="px-3 py-2.5 font-semibold">Interface</th>
            <th class="px-3 py-2.5 font-semibold">Lines</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="rec in filtered"
            :key="rec.id"
            class="border-b border-border/50 hover:bg-bg-hover transition-colors duration-100"
          >
            <td class="px-5 py-2.5 font-mono text-fg-muted whitespace-nowrap">{{ formatTime(rec.start_time) }}</td>
            <td class="px-3 py-2.5">
              <span class="font-mono text-fg">{{ rec.runbook_id }}</span>
            </td>
            <td class="px-3 py-2.5">
              <span
                :class="statusClass(rec.status)"
                class="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-semibold rounded-full"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-current"></span>
                {{ rec.status }}
              </span>
            </td>
            <td class="px-3 py-2.5 font-mono text-fg-muted">{{ rec.duration || "–" }}</td>
            <td class="px-3 py-2.5">
              <span class="text-[10px] font-mono px-1.5 py-0.5 bg-bg-element rounded text-fg-subtle">
                {{ interfaceBadge(rec.interface) }}
              </span>
            </td>
            <td class="px-3 py-2.5 font-mono text-fg-muted">{{ rec.output_lines }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
