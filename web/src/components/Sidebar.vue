<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { fetchThemes, setTheme } from "../lib/api";
import type { Catalog, RunbookSummary } from "../lib/types";

const props = defineProps<{
  catalogs: Catalog[];
  loading: boolean;
}>();

const emit = defineEmits<{
  (e: "themeChanged", colors: Record<string, string>): void;
  (e: "update:open", value: boolean): void;
}>();

const router = useRouter();
const route = useRoute();
const collapsed = ref<Record<string, boolean>>({});
const filter = ref("");
const searchEl = ref<HTMLInputElement | null>(null);
const focusedIndex = ref(-1);

const themeNames = ref<string[]>([]);
const activeTheme = ref("");
const showThemeMenu = ref(false);
const themeMenuEl = ref<HTMLElement | null>(null);

// Flat list of visible runbooks for keyboard navigation.
const visibleRunbooks = computed<RunbookSummary[]>(() => {
  const result: RunbookSummary[] = [];
  for (const cat of props.catalogs) {
    if (collapsed.value[cat.name]) continue;
    for (const rb of filteredRunbooks(cat)) {
      result.push(rb);
    }
  }
  return result;
});

async function loadThemes() {
  const data = await fetchThemes();
  themeNames.value = data.themes;
  activeTheme.value = data.active;
}

async function selectTheme(name: string) {
  showThemeMenu.value = false;
  activeTheme.value = name;
  const result = await setTheme(name);
  emit("themeChanged", result.colors);
}

function onClickOutside(e: MouseEvent) {
  if (themeMenuEl.value && !themeMenuEl.value.contains(e.target as Node)) {
    showThemeMenu.value = false;
  }
}

// Global keyboard handler.
function onKeydown(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement)?.tagName;
  const isInput = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
  const isSearchFocused = document.activeElement === searchEl.value;

  // "/" focuses search (unless already typing in an input).
  if (e.key === "/" && !isInput) {
    e.preventDefault();
    searchEl.value?.focus();
    focusedIndex.value = -1;
    return;
  }

  // The rest only apply when search is focused.
  if (!isSearchFocused) return;

  if (e.key === "Escape") {
    searchEl.value?.blur();
    focusedIndex.value = -1;
    return;
  }

  const list = visibleRunbooks.value;
  if (list.length === 0) return;

  if (e.key === "ArrowDown" || (e.key === "j" && e.ctrlKey)) {
    e.preventDefault();
    focusedIndex.value = Math.min(focusedIndex.value + 1, list.length - 1);
    return;
  }

  if (e.key === "ArrowUp" || (e.key === "k" && e.ctrlKey)) {
    e.preventDefault();
    focusedIndex.value = Math.max(focusedIndex.value - 1, -1);
    return;
  }

  if (e.key === "Enter" && focusedIndex.value >= 0 && focusedIndex.value < list.length) {
    e.preventDefault();
    navigateTo(`/runbook/${list[focusedIndex.value].id}`);
    searchEl.value?.blur();
    focusedIndex.value = -1;
    return;
  }
}

onMounted(() => {
  loadThemes();
  document.addEventListener("click", onClickOutside);
  document.addEventListener("keydown", onKeydown);
});
onUnmounted(() => {
  document.removeEventListener("click", onClickOutside);
  document.removeEventListener("keydown", onKeydown);
});

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

function isFocused(rbId: string): boolean {
  const idx = focusedIndex.value;
  if (idx < 0) return false;
  const list = visibleRunbooks.value;
  return idx < list.length && list[idx].id === rbId;
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

function navigateTo(path: string) {
  router.push(path);
  emit("update:open", false);
}
</script>

<template>
  <aside
    class="w-[280px] min-w-[280px] border-r border-border bg-bg-panel flex flex-col h-full overflow-hidden"
  >
    <!-- Header -->
    <div class="px-5 pt-5 pb-4 border-b border-border flex items-center justify-between">
      <button
        @click="navigateTo('/')"
        class="text-[15px] font-bold text-fg flex items-center gap-2 bg-transparent border-none cursor-pointer p-0 hover:opacity-80 transition-opacity duration-150"
      >
        <span class="text-primary font-mono text-[15px] font-bold tracking-tight">dops</span>
        runbooks
      </button>
      <div class="flex items-center gap-2.5">
        <!-- History -->
        <button
          @click="navigateTo('/history')"
          class="text-fg-subtle hover:text-fg-muted transition-colors duration-150 p-0 bg-transparent border-none cursor-pointer"
          title="Execution history"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M1.5 8a6.5 6.5 0 1113 0 6.5 6.5 0 01-13 0zM8 0a8 8 0 100 16A8 8 0 008 0zm.5 4.75a.75.75 0 00-1.5 0v3.5a.75.75 0 00.37.65l2.5 1.5a.75.75 0 00.76-1.3L8.5 7.87V4.75z"/></svg>
        </button>
        <!-- Theme selector -->
        <div ref="themeMenuEl" class="relative">
          <button
            @click.stop="showThemeMenu = !showThemeMenu"
            class="text-fg-subtle hover:text-fg-muted transition-colors duration-150 p-0 bg-transparent border-none cursor-pointer"
            title="Change theme"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M8 1a7 7 0 100 14A7 7 0 008 1zM0 8a8 8 0 1116 0A8 8 0 010 8zm8-5a5 5 0 00-3.544 8.544L8 8V3z"/></svg>
          </button>

          <!-- Dropdown -->
          <div
            v-if="showThemeMenu"
            class="absolute right-0 top-8 w-[180px] max-h-[320px] overflow-y-auto bg-bg-panel border border-border rounded-lg shadow-xl z-50 py-1"
          >
            <button
              v-for="name in themeNames"
              :key="name"
              @click="selectTheme(name)"
              :class="name === activeTheme
                ? 'bg-primary-muted text-primary'
                : 'text-fg-muted hover:bg-bg-hover hover:text-fg'"
              class="flex items-center gap-2 w-full text-left px-3 py-1.5 text-[13px] cursor-pointer transition-all duration-100 border-none bg-transparent"
            >
              <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="name === activeTheme ? 'bg-primary' : 'bg-transparent'"></span>
              {{ name }}
            </button>
          </div>
        </div>
        <a
          href="https://rundops.dev/"
          target="_blank"
          rel="noopener noreferrer"
          class="text-fg-subtle hover:text-fg-muted transition-colors duration-150"
          title="Documentation"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M0 1.75A.75.75 0 01.75 1h4.253c1.227 0 2.317.59 3 1.501A3.744 3.744 0 0111.006 1h4.245a.75.75 0 01.75.75v10.5a.75.75 0 01-.75.75h-4.507a2.25 2.25 0 00-1.591.659l-.622.621a.75.75 0 01-1.06 0l-.623-.621A2.25 2.25 0 005.258 13H.75a.75.75 0 01-.75-.75zm7.251 10.324l.004-5.073-.002-2.253A2.25 2.25 0 005.003 2.5H1.5v9h3.757a3.75 3.75 0 011.994.574zM8.755 4.75l-.004 7.322a3.752 3.752 0 011.992-.572H14.5v-9h-3.495a2.25 2.25 0 00-2.25 2.25z"/></svg>
        </a>
        <a
          href="https://github.com/rundops/dops"
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
        ref="searchEl"
        v-model="filter"
        type="text"
        placeholder="Search runbooks\u2026"
        class="w-full px-3 py-[7px] text-[13px] bg-bg border border-border rounded-md text-fg placeholder-fg-subtle focus:border-border-active focus:outline-none transition-colors duration-150"
        @input="focusedIndex = -1"
      />
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto px-2 pt-1 pb-4">
      <div v-if="loading" class="text-fg-muted text-sm p-2">Loading...</div>

      <div v-for="cat in catalogs" :key="cat.name" v-show="!filter || filteredRunbooks(cat).length" class="mb-1">
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
            @click="navigateTo(`/runbook/${rb.id}`)"
            :class="[
              isActive(rb.id)
                ? 'bg-primary-muted text-primary'
                : isFocused(rb.id)
                  ? 'bg-bg-hover text-fg'
                  : 'text-fg-muted hover:bg-bg-hover hover:text-fg'
            ]"
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
