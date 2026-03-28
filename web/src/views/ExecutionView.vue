<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from "vue";
import { streamExecution, cancelExecution } from "../lib/api";
import { AnsiUp } from "ansi_up";

const props = defineProps<{ id: string }>();

const lines = ref<string[]>([]);
const status = ref<"running" | "success" | "error">("running");
const statusMessage = ref("");
const outputEl = ref<HTMLElement | null>(null);

const ansi = new AnsiUp();
ansi.use_classes = false;

let eventSource: EventSource | null = null;

function scrollToBottom() {
  nextTick(() => {
    if (outputEl.value) {
      outputEl.value.scrollTop = outputEl.value.scrollHeight;
    }
  });
}

onMounted(() => {
  eventSource = streamExecution(
    props.id,
    (line) => {
      lines.value.push(line);
      scrollToBottom();
    },
    (msg) => {
      if (msg.startsWith("error")) {
        status.value = "error";
        statusMessage.value = msg;
      } else {
        status.value = "success";
        statusMessage.value = "Completed successfully";
      }
    },
    () => {
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
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-between p-4 border-b border-border">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-bold text-fg">Execution</h2>
        <span
          :class="{
            'text-success': status === 'success',
            'text-error': status === 'error',
            'text-primary': status === 'running',
          }"
          class="text-sm font-medium"
        >
          {{ status === "running" ? "Running..." : statusMessage }}
        </span>
      </div>
      <button
        v-if="status === 'running'"
        @click="cancel"
        class="px-3 py-1 text-sm bg-error/20 text-error rounded hover:bg-error/30"
      >
        Cancel
      </button>
    </div>

    <!-- Output -->
    <div
      ref="outputEl"
      class="flex-1 overflow-auto p-4 bg-bg font-mono text-sm leading-relaxed"
    >
      <div
        v-for="(line, i) in lines"
        :key="i"
        v-html="renderAnsi(line)"
        class="whitespace-pre-wrap"
      />
      <div v-if="lines.length === 0 && status === 'running'" class="text-fg-muted">
        Waiting for output...
      </div>
    </div>
  </div>
</template>
