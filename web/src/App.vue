<script setup lang="ts">
import { ref, onMounted } from "vue";
import Sidebar from "./components/Sidebar.vue";
import ToastContainer from "./components/ToastContainer.vue";
import { fetchCatalogs, fetchTheme } from "./lib/api";
import type { Catalog } from "./lib/types";

const catalogs = ref<Catalog[]>([]);
const loading = ref(true);

// Theme token name → CSS custom property mapping.
const themeMap: Record<string, string> = {
  background: "--dops-background",
  backgroundPanel: "--dops-backgroundPanel",
  backgroundElement: "--dops-backgroundElement",
  backgroundHover: "--dops-backgroundHover",
  text: "--dops-text",
  textMuted: "--dops-textMuted",
  textSubtle: "--dops-textSubtle",
  primary: "--dops-primary",
  primaryMuted: "--dops-primaryMuted",
  border: "--dops-border",
  borderActive: "--dops-borderActive",
  success: "--dops-success",
  successMuted: "--dops-successMuted",
  warning: "--dops-warning",
  warningMuted: "--dops-warningMuted",
  error: "--dops-error",
  errorMuted: "--dops-errorMuted",
};

function applyTheme(colors: Record<string, string>) {
  const root = document.documentElement;
  for (const [token, cssVar] of Object.entries(themeMap)) {
    const value = colors[token];
    if (value && value !== "none") {
      root.style.setProperty(cssVar, value);
    }
  }
}

onMounted(async () => {
  // Load theme and catalogs in parallel.
  const [themeResult] = await Promise.allSettled([
    fetchTheme().then((t) => applyTheme(t.colors)),
    fetchCatalogs()
      .then((c) => (catalogs.value = c))
      .finally(() => (loading.value = false)),
  ]);
  if (themeResult.status === "rejected") {
    console.error("Failed to load theme:", themeResult.reason);
  }
});
</script>

<template>
  <div class="flex h-screen">
    <Sidebar :catalogs="catalogs" :loading="loading" />
    <main class="flex-1 overflow-hidden">
      <router-view />
    </main>
    <ToastContainer />
  </div>
</template>
