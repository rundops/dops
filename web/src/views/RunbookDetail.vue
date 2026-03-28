<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { fetchRunbook, executeRunbook } from "../lib/api";
import type { RunbookDetail } from "../lib/types";

const props = defineProps<{ id: string }>();
const router = useRouter();

const runbook = ref<RunbookDetail | null>(null);
const params = ref<Record<string, string>>({});
const loading = ref(true);
const executing = ref(false);
const error = ref("");

async function load() {
  loading.value = true;
  error.value = "";
  try {
    runbook.value = await fetchRunbook(props.id);
    // Initialize params with defaults.
    params.value = {};
    for (const p of runbook.value.parameters) {
      if (p.default !== undefined && p.default !== null) {
        params.value[p.name] = String(p.default);
      }
    }
  } catch (e) {
    error.value = String(e);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
watch(() => props.id, load);

async function execute() {
  if (!runbook.value) return;
  executing.value = true;
  error.value = "";
  try {
    const execId = await executeRunbook(runbook.value.id, params.value);
    router.push(`/execute/${execId}`);
  } catch (e) {
    error.value = String(e);
    executing.value = false;
  }
}

function riskBadge(level: string): string {
  switch (level) {
    case "critical":
      return "bg-error/20 text-error";
    case "high":
      return "bg-warning/20 text-warning";
    case "medium":
      return "bg-primary/20 text-primary";
    default:
      return "bg-bg-element text-fg-muted";
  }
}
</script>

<template>
  <div v-if="loading" class="text-fg-muted">Loading...</div>
  <div v-else-if="error" class="text-error">{{ error }}</div>
  <div v-else-if="runbook" class="max-w-2xl">
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-xl font-bold text-fg">{{ runbook.name }}</h2>
      <span
        v-if="runbook.risk_level"
        :class="riskBadge(runbook.risk_level)"
        class="px-2 py-0.5 rounded text-xs font-medium"
      >
        {{ runbook.risk_level }}
      </span>
      <span class="text-fg-muted text-xs">v{{ runbook.version }}</span>
    </div>

    <p class="text-fg-muted mb-6">{{ runbook.description }}</p>

    <div v-if="runbook.aliases?.length" class="mb-4">
      <span class="text-fg-muted text-sm">Aliases: </span>
      <span
        v-for="alias in runbook.aliases"
        :key="alias"
        class="inline-block bg-bg-element text-fg-muted text-xs px-1.5 py-0.5 rounded mr-1"
      >
        {{ alias }}
      </span>
    </div>

    <!-- Parameter Form -->
    <form v-if="runbook.parameters.length" @submit.prevent="execute" class="space-y-4">
      <div v-for="p in runbook.parameters" :key="p.name">
        <label class="block text-sm font-medium text-fg mb-1">
          {{ p.name }}
          <span v-if="p.required" class="text-error">*</span>
        </label>
        <p v-if="p.description" class="text-xs text-fg-muted mb-1">{{ p.description }}</p>

        <!-- Select -->
        <select
          v-if="p.type === 'select' && p.options"
          v-model="params[p.name]"
          class="w-full px-2 py-1.5 text-sm bg-bg-element border border-border rounded text-fg focus:border-border-active focus:outline-none"
        >
          <option v-for="opt in p.options" :key="opt" :value="opt">{{ opt }}</option>
        </select>

        <!-- Boolean -->
        <div v-else-if="p.type === 'boolean'" class="flex gap-4">
          <label class="flex items-center gap-1 text-sm text-fg">
            <input type="radio" :name="p.name" value="true" v-model="params[p.name]" />
            Yes
          </label>
          <label class="flex items-center gap-1 text-sm text-fg">
            <input type="radio" :name="p.name" value="false" v-model="params[p.name]" />
            No
          </label>
        </div>

        <!-- Secret -->
        <input
          v-else-if="p.secret"
          type="password"
          v-model="params[p.name]"
          :placeholder="p.name"
          class="w-full px-2 py-1.5 text-sm bg-bg-element border border-border rounded text-fg placeholder-fg-muted focus:border-border-active focus:outline-none"
        />

        <!-- Text (default) -->
        <input
          v-else
          type="text"
          v-model="params[p.name]"
          :placeholder="p.default !== undefined ? String(p.default) : p.name"
          class="w-full px-2 py-1.5 text-sm bg-bg-element border border-border rounded text-fg placeholder-fg-muted focus:border-border-active focus:outline-none"
        />
      </div>

      <button
        type="submit"
        :disabled="executing"
        class="px-4 py-2 bg-primary text-bg font-medium rounded hover:opacity-90 disabled:opacity-50"
      >
        {{ executing ? "Executing..." : "Execute" }}
      </button>
    </form>

    <!-- No params -->
    <div v-else>
      <button
        @click="execute"
        :disabled="executing"
        class="px-4 py-2 bg-primary text-bg font-medium rounded hover:opacity-90 disabled:opacity-50"
      >
        {{ executing ? "Executing..." : "Execute" }}
      </button>
    </div>
  </div>
</template>
