<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRouter } from "vue-router";
import { fetchRunbook, executeRunbook } from "../lib/api";
import { useToast } from "../lib/toast";
import type { RunbookDetail, Parameter } from "../lib/types";

const props = defineProps<{ id: string }>();
const router = useRouter();
const toast = useToast();

const runbook = ref<RunbookDetail | null>(null);
const params = ref<Record<string, string>>({});
const loading = ref(true);
const executing = ref(false);
const error = ref("");
const showConfirm = ref(false);
const confirmInput = ref("");
const showSaved = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  showSaved.value = false;
  try {
    runbook.value = await fetchRunbook(props.id);
    params.value = {};
    const saved = runbook.value.saved_values || {};

    for (const p of runbook.value.parameters) {
      // Saved values take priority, then defaults.
      if (saved[p.name] !== undefined) {
        params.value[p.name] = saved[p.name];
      } else if (p.default !== undefined && p.default !== null) {
        params.value[p.name] = String(p.default);
      }
    }
  } catch (e) {
    error.value = String(e);
  } finally {
    loading.value = false;
  }
}

function hasSavedValue(p: Parameter): boolean {
  if (!runbook.value?.saved_values) return false;
  return runbook.value.saved_values[p.name] !== undefined;
}

const unsavedParams = computed<Parameter[]>(() => {
  if (!runbook.value) return [];
  return runbook.value.parameters.filter((p) => !hasSavedValue(p));
});

const savedParams = computed<Parameter[]>(() => {
  if (!runbook.value) return [];
  return runbook.value.parameters.filter((p) => hasSavedValue(p));
});

function onKeydown(e: KeyboardEvent) {
  if (e.key === "Escape" && showConfirm.value) {
    showConfirm.value = false;
  }
}

onMounted(() => {
  load();
  document.addEventListener("keydown", onKeydown);
});
onUnmounted(() => {
  document.removeEventListener("keydown", onKeydown);
});
watch(() => props.id, load);

function needsConfirmation(): boolean {
  if (!runbook.value) return false;
  return runbook.value.risk_level === "high" || runbook.value.risk_level === "critical";
}

function attemptExecute() {
  if (needsConfirmation()) {
    showConfirm.value = true;
    confirmInput.value = "";
    return;
  }
  execute();
}

async function execute() {
  if (!runbook.value) return;
  showConfirm.value = false;
  executing.value = true;
  error.value = "";
  try {
    const execId = await executeRunbook(runbook.value.id, params.value);
    router.push({ path: `/execute/${execId}`, query: { name: runbook.value.name } });
  } catch (e) {
    error.value = String(e);
    toast.error(String(e));
    executing.value = false;
  }
}

function confirmValid(): boolean {
  if (!runbook.value) return false;
  if (runbook.value.risk_level === "critical") {
    return confirmInput.value === "CONFIRM";
  }
  // High risk: just needs acknowledgment (click confirm).
  return true;
}

function riskBadgeClass(level: string): string {
  switch (level) {
    case "critical":
      return "bg-error text-white";
    case "high":
      return "bg-warning text-black";
    case "medium":
      return "bg-primary text-black";
    default:
      return "bg-bg-element text-fg-muted";
  }
}

function catalogFromId(id: string): string {
  const parts = id.split("/");
  return parts.length > 1 ? parts[0] : "";
}

function toggleMultiSelect(paramName: string, option: string) {
  const current = params.value[paramName] || "";
  const selected = current ? current.split(", ") : [];
  const idx = selected.indexOf(option);
  if (idx >= 0) {
    selected.splice(idx, 1);
  } else {
    selected.push(option);
  }
  params.value[paramName] = selected.join(", ");
}

function isMultiSelected(paramName: string, option: string): boolean {
  const current = params.value[paramName] || "";
  return current.split(", ").includes(option);
}
</script>

<template>
  <div class="h-full overflow-y-auto p-10">
    <!-- Loading skeleton -->
    <div v-if="loading" class="max-w-[640px] animate-pulse space-y-4">
      <div class="flex items-center gap-3">
        <div class="h-6 w-48 bg-bg-element rounded"></div>
        <div class="h-5 w-16 bg-bg-element rounded-full"></div>
      </div>
      <div class="h-4 w-full bg-bg-element rounded"></div>
      <div class="h-4 w-3/4 bg-bg-element rounded"></div>
      <div class="space-y-3 mt-6">
        <div class="h-4 w-24 bg-bg-element rounded"></div>
        <div class="h-8 w-full bg-bg-element rounded"></div>
        <div class="h-4 w-24 bg-bg-element rounded"></div>
        <div class="h-8 w-full bg-bg-element rounded"></div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="text-error">{{ error }}</div>

    <!-- Runbook content -->
    <div v-else-if="runbook" class="max-w-[640px]">
      <!-- Header -->
      <div class="mb-8">
        <!-- Title row -->
        <div class="flex items-center gap-3 mb-2">
          <h2 class="text-[22px] font-bold tracking-tight text-fg">{{ runbook.name }}</h2>
          <span
            v-if="runbook.risk_level"
            :class="riskBadgeClass(runbook.risk_level)"
            class="inline-flex items-center px-2 py-0.5 text-[11px] font-semibold rounded-full uppercase tracking-wide"
          >
            {{ runbook.risk_level }}
          </span>
        </div>

        <!-- Meta line -->
        <div class="flex items-center gap-4 text-[13px] text-fg-muted">
          <span>v{{ runbook.version }}</span>
          <span v-if="catalogFromId(runbook.id)" class="text-fg-subtle">&middot;</span>
          <span v-if="catalogFromId(runbook.id)">{{ catalogFromId(runbook.id) }}</span>
          <span class="text-fg-subtle">&middot;</span>
          <span>{{ runbook.parameters.length }} {{ runbook.parameters.length === 1 ? 'parameter' : 'parameters' }}</span>
        </div>

        <!-- Description -->
        <p class="mt-3 text-sm leading-relaxed text-fg-muted">{{ runbook.description }}</p>

        <!-- Aliases -->
        <div v-if="runbook.aliases?.length" class="flex gap-1.5 mt-3">
          <span
            v-for="alias in runbook.aliases"
            :key="alias"
            class="font-mono text-xs px-2 py-0.5 bg-bg-element rounded text-fg-muted"
          >
            {{ alias }}
          </span>
        </div>
      </div>

      <!-- Parameter Form -->
      <form
        v-if="runbook.parameters.length"
        @submit.prevent="attemptExecute"
        class="border-t border-border pt-6"
      >
        <!-- Unsaved parameters (always visible) -->
        <div v-if="unsavedParams.length" class="text-[13px] font-semibold uppercase tracking-wide text-fg-muted mb-5">
          Parameters
        </div>

        <div v-for="p in unsavedParams" :key="p.name" class="mb-5">
          <!-- Label -->
          <label class="flex items-center gap-1.5 text-[13px] font-semibold text-fg mb-1">
            {{ p.name }}
            <span v-if="p.required" class="text-error font-normal">*</span>
          </label>

          <!-- Hint -->
          <div v-if="p.description" class="text-xs text-fg-muted mb-2">{{ p.description }}</div>

          <!-- Select -->
          <select
            v-if="p.type === 'select' && p.options"
            v-model="params[p.name]"
            class="select-arrow w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
          >
            <option v-for="opt in p.options" :key="opt" :value="opt">{{ opt }}</option>
          </select>

          <!-- Multi-select chips -->
          <div v-else-if="p.type === 'multi_select' && p.options" class="flex flex-wrap gap-1.5">
            <button
              v-for="opt in p.options"
              :key="opt"
              type="button"
              @click="toggleMultiSelect(p.name, opt)"
              :class="isMultiSelected(p.name, opt)
                ? 'bg-primary-muted border-primary text-primary'
                : 'border-border text-fg-muted hover:border-fg-subtle hover:text-fg'"
              class="inline-flex items-center gap-1 px-3 py-[5px] text-[13px] border rounded-full cursor-pointer transition-all duration-150"
            >
              <span v-if="isMultiSelected(p.name, opt)" class="text-[11px]">&#10003;</span>
              {{ opt }}
            </button>
          </div>

          <!-- Boolean toggle -->
          <div
            v-else-if="p.type === 'boolean'"
            class="flex gap-0.5 bg-bg border border-border rounded-lg p-0.5 w-fit"
          >
            <button
              type="button"
              @click="params[p.name] = 'true'"
              :class="params[p.name] === 'true' ? 'bg-bg-element text-fg' : 'text-fg-muted'"
              class="px-4 py-1.5 text-[13px] rounded-md border-none cursor-pointer transition-all duration-150"
            >
              Yes
            </button>
            <button
              type="button"
              @click="params[p.name] = 'false'"
              :class="params[p.name] === 'false' ? 'bg-bg-element text-fg' : 'text-fg-muted'"
              class="px-4 py-1.5 text-[13px] rounded-md border-none cursor-pointer transition-all duration-150"
            >
              No
            </button>
          </div>

          <!-- Secret -->
          <input
            v-else-if="p.secret"
            type="password"
            v-model="params[p.name]"
            :placeholder="p.name"
            class="w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
          />

          <!-- Text (default) -->
          <input
            v-else
            type="text"
            v-model="params[p.name]"
            :placeholder="p.default !== undefined ? String(p.default) : p.name"
            class="w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
          />
        </div>

        <!-- Saved values (collapsible) -->
        <div v-if="savedParams.length" class="mt-6">
          <button
            type="button"
            @click="showSaved = !showSaved"
            class="flex items-center gap-2 text-[13px] font-semibold uppercase tracking-wide text-fg-muted mb-4 cursor-pointer hover:text-fg transition-colors duration-150 bg-transparent border-none p-0"
          >
            <span class="text-[10px] transition-transform duration-150" :class="showSaved ? 'rotate-90' : ''">&#9654;</span>
            Saved values
            <span class="text-[11px] font-normal normal-case tracking-normal px-1.5 py-0.5 bg-bg-element rounded-full">
              {{ savedParams.length }} applied
            </span>
          </button>

          <div v-if="showSaved" class="space-y-5">
            <div v-for="p in savedParams" :key="p.name" class="mb-5">
              <!-- Label with saved badge -->
              <label class="flex items-center gap-1.5 text-[13px] font-semibold text-fg mb-1">
                {{ p.name }}
                <span v-if="p.required" class="text-error font-normal">*</span>
                <span class="text-[10px] font-normal text-success px-1.5 py-0.5 bg-success-muted rounded-full ml-1">saved</span>
              </label>

              <!-- Hint -->
              <div v-if="p.description" class="text-xs text-fg-muted mb-2">{{ p.description }}</div>

              <!-- Select -->
              <select
                v-if="p.type === 'select' && p.options"
                v-model="params[p.name]"
                class="select-arrow w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
              >
                <option v-for="opt in p.options" :key="opt" :value="opt">{{ opt }}</option>
              </select>

              <!-- Multi-select chips -->
              <div v-else-if="p.type === 'multi_select' && p.options" class="flex flex-wrap gap-1.5">
                <button
                  v-for="opt in p.options"
                  :key="opt"
                  type="button"
                  @click="toggleMultiSelect(p.name, opt)"
                  :class="isMultiSelected(p.name, opt)
                    ? 'bg-primary-muted border-primary text-primary'
                    : 'border-border text-fg-muted hover:border-fg-subtle hover:text-fg'"
                  class="inline-flex items-center gap-1 px-3 py-[5px] text-[13px] border rounded-full cursor-pointer transition-all duration-150"
                >
                  <span v-if="isMultiSelected(p.name, opt)" class="text-[11px]">&#10003;</span>
                  {{ opt }}
                </button>
              </div>

              <!-- Boolean toggle -->
              <div
                v-else-if="p.type === 'boolean'"
                class="flex gap-0.5 bg-bg border border-border rounded-lg p-0.5 w-fit"
              >
                <button
                  type="button"
                  @click="params[p.name] = 'true'"
                  :class="params[p.name] === 'true' ? 'bg-bg-element text-fg' : 'text-fg-muted'"
                  class="px-4 py-1.5 text-[13px] rounded-md border-none cursor-pointer transition-all duration-150"
                >
                  Yes
                </button>
                <button
                  type="button"
                  @click="params[p.name] = 'false'"
                  :class="params[p.name] === 'false' ? 'bg-bg-element text-fg' : 'text-fg-muted'"
                  class="px-4 py-1.5 text-[13px] rounded-md border-none cursor-pointer transition-all duration-150"
                >
                  No
                </button>
              </div>

              <!-- Secret (show masked, not editable from saved) -->
              <input
                v-else-if="p.secret"
                type="password"
                v-model="params[p.name]"
                :placeholder="p.name"
                class="w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
              />

              <!-- Text (default) -->
              <input
                v-else
                type="text"
                v-model="params[p.name]"
                :placeholder="p.default !== undefined ? String(p.default) : p.name"
                class="w-full px-3 py-2 text-sm font-mono bg-bg-element border border-border rounded-lg text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none focus:ring-3 focus:ring-primary-muted transition-colors duration-150"
              />
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="mt-8 pt-6 border-t border-border flex gap-3 items-center">
          <button
            type="submit"
            :disabled="executing"
            class="px-6 py-2.5 text-sm font-semibold bg-primary text-black rounded-lg cursor-pointer hover:opacity-90 disabled:opacity-50 flex items-center gap-2 transition-opacity duration-150"
          >
            <span>&#9654;</span>
            {{ executing ? "Executing..." : "Execute" }}
          </button>
          <span
            v-if="runbook.risk_level === 'high' || runbook.risk_level === 'critical'"
            class="text-xs flex items-center gap-1.5"
            :class="runbook.risk_level === 'critical' ? 'text-error' : 'text-warning'"
          >
            &#9888; {{ runbook.risk_level === 'critical' ? 'Critical' : 'High' }} risk — requires confirmation
          </span>
        </div>
      </form>

      <!-- No params -->
      <div v-else class="mt-8 pt-6 border-t border-border flex gap-3 items-center">
        <button
          @click="attemptExecute"
          :disabled="executing"
          class="px-6 py-2.5 text-sm font-semibold bg-primary text-black rounded-lg cursor-pointer hover:opacity-90 disabled:opacity-50 flex items-center gap-2 transition-opacity duration-150"
        >
          <span>&#9654;</span>
          {{ executing ? "Executing..." : "Execute" }}
        </button>
        <span
          v-if="runbook.risk_level === 'high' || runbook.risk_level === 'critical'"
          class="text-xs flex items-center gap-1.5"
          :class="runbook.risk_level === 'critical' ? 'text-error' : 'text-warning'"
        >
          &#9888; {{ runbook.risk_level === 'critical' ? 'Critical' : 'High' }} risk — requires confirmation
        </span>
      </div>

      <!-- Risk Confirmation Dialog -->
      <div
        v-if="showConfirm"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
        @click.self="showConfirm = false"
      >
        <div class="bg-bg-panel border border-border rounded-xl p-7 w-[420px] mx-4 shadow-2xl">
          <!-- Icon -->
          <div
            :class="runbook.risk_level === 'critical'
              ? 'bg-error-muted text-error'
              : 'bg-warning-muted text-warning'"
            class="w-10 h-10 rounded-[10px] flex items-center justify-center text-xl mb-4"
          >
            {{ runbook.risk_level === 'critical' ? '\u2715' : '\u26A0' }}
          </div>

          <!-- Title -->
          <h3 class="text-base font-bold text-fg mb-2">
            {{ runbook.risk_level === 'critical' ? 'Critical Operation' : 'Confirm Execution' }}
          </h3>

          <!-- Description -->
          <p class="text-sm leading-relaxed text-fg-muted mb-5">
            <template v-if="runbook.risk_level === 'critical'">
              You are about to execute <strong class="text-fg">{{ runbook.name }}</strong>.
              This is a <strong class="text-error">critical</strong> operation that cannot be easily reversed.
              Type <code class="font-mono text-[13px] bg-bg-element px-1.5 rounded text-fg">CONFIRM</code> to proceed.
            </template>
            <template v-else>
              You are about to execute <strong class="text-fg">{{ runbook.name }}</strong>.
              This is a high-risk operation that may affect running services.
            </template>
          </p>

          <!-- Confirm input for critical -->
          <input
            v-if="runbook.risk_level === 'critical'"
            v-model="confirmInput"
            type="text"
            placeholder="Type CONFIRM"
            class="w-full px-3 py-2 text-sm font-mono bg-bg border border-border rounded-lg text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none mb-5 transition-colors duration-150"
            @keydown.enter="confirmValid() && execute()"
          />

          <!-- Dialog actions -->
          <div class="flex gap-2.5 justify-end">
            <button
              @click="showConfirm = false"
              class="px-[18px] py-2 text-[13px] font-semibold rounded-md border border-border text-fg-muted hover:border-fg-subtle hover:text-fg bg-transparent cursor-pointer transition-all duration-150"
            >
              Cancel
            </button>
            <button
              @click="execute"
              :disabled="!confirmValid()"
              :class="runbook.risk_level === 'critical'
                ? 'bg-error text-white hover:opacity-90'
                : 'bg-warning text-black hover:opacity-90'"
              class="px-[18px] py-2 text-[13px] font-semibold rounded-md border-none cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed transition-all duration-150"
            >
              Execute
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
