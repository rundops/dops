<script setup lang="ts">
import { ref, onMounted } from "vue";
import Sidebar from "./components/Sidebar.vue";
import type { Catalog } from "./lib/types";

const catalogs = ref<Catalog[]>([]);
const loading = ref(true);

onMounted(async () => {
  try {
    const res = await fetch("/api/catalogs");
    catalogs.value = await res.json();
  } catch (e) {
    console.error("Failed to load catalogs:", e);
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="flex h-screen">
    <Sidebar :catalogs="catalogs" :loading="loading" />
    <main class="flex-1 overflow-auto p-6">
      <router-view />
    </main>
  </div>
</template>
