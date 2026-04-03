<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import { streamExecution, cancelExecution } from "../lib/api";
import { AnsiUp } from "ansi_up";

const props = defineProps<{ id: string }>();
const router = useRouter();
const route = useRoute();

const lines = ref<string[]>([]);
const status = ref<"running" | "success" | "error">("running");
const statusMessage = ref("");
const outputEl = ref<HTMLElement | null>(null);
const startTime = ref(Date.now());
const endTime = ref<number | null>(null);

const ansi = new AnsiUp();
ansi.use_classes = false;

let eventSource: EventSource | null = null;

const duration = computed(() => {
  const end = endTime.value ?? Date.now();
  const seconds = ((end - startTime.value) / 1000).toFixed(1);
  return `${seconds}s`;
});

const isComplete = computed(() => status.value !== "running");

const runbookName = computed(() => {
  return (route.query.name as string) || props.id;
});

function scrollToBottom() {
  nextTick(() => {
    if (outputEl.value) {
      outputEl.value.scrollTop = outputEl.value.scrollHeight;
    }
  });
}

onMounted(() => {
  startTime.value = Date.now();
  eventSource = streamExecution(
    props.id,
    (line) => {
      lines.value.push(line);
      scrollToBottom();
    },
    (msg) => {
      endTime.value = Date.now();
      if (msg.startsWith("error")) {
        status.value = "error";
        statusMessage.value = msg;
      } else {
        status.value = "success";
        statusMessage.value = "Completed successfully";
      }
    },
    () => {
      endTime.value = Date.now();
      status.value = "error";
      statusMessage.value = "Connection lost";
    }
  );
});

onUnmounted(() => {
  eventSource?.close();
});

async function cancel() {
  await cancelExecution(props.id);
}

function renderAnsi(text: string): string {
  return ansi.ansi_to_html(text);
}

function statusPillClass(): string {
  switch (status.value) {
    case "running":
      return "bg-primary-muted text-primary";
    case "success":
      return "bg-success-muted text-success";
    case "error":
      return "bg-error-muted text-error";
  }
}

function statusLabel(): string {
  switch (status.value) {
    case "running":
      return "Running";
    case "success":
      return "Completed";
    case "error":
      return "Failed";
  }
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header bar -->
    <div class="px-6 py-4 border-b border-border flex items-center justify-between bg-bg-panel">
      <div class="flex items-center gap-4">
        <span class="font-mono text-[15px] font-bold text-fg">{{ runbookName }}</span>
        <span
          :class="statusPillClass()"
          class="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-full"
        >
          <span
            class="w-1.5 h-1.5 rounded-full bg-current"
            :class="{ 'animate-pulse-dot': status === 'running' }"
          ></span>
          {{ statusLabel() }}
        </span>
        <span v-if="isComplete" class="text-fg-muted font-mono text-xs">{{ duration }}</span>
      </div>
      <button
        v-if="status === 'running'"
        @click="cancel"
        class="px-3.5 py-1.5 text-[13px] font-medium border border-error rounded-md bg-transparent text-error cursor-pointer hover:bg-error-muted transition-all duration-150"
      >
        Cancel
      </button>
      <button
        v-else
        @click="router.back()"
        class="px-3.5 py-1.5 text-[13px] font-medium border border-border rounded-md bg-transparent text-fg-muted cursor-pointer hover:border-fg-subtle hover:text-fg transition-all duration-150"
      >
        &larr; Back
      </button>
    </div>

    <!-- Log output -->
    <div
      ref="outputEl"
      class="flex-1 overflow-y-auto px-6 py-4 bg-bg font-mono text-[13px] leading-[1.7]"
    >
      <div
        v-for="(line, i) in lines"
        :key="i"
        class="flex gap-3 whitespace-pre-wrap break-all"
      >
        <span class="text-fg-subtle select-none text-right min-w-[28px] shrink-0">{{ i + 1 }}</span>
        <span class="text-fg-muted" v-html="renderAnsi(line)"></span>
      </div>
      <div v-if="lines.length === 0 && status === 'running'" class="text-fg-muted">
        Waiting for output...
      </div>
    </div>

  </div>
</template>
