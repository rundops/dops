<script setup lang="ts">
import { useToast } from "../lib/toast";

const { toasts } = useToast();

function toastClass(type: string): string {
  switch (type) {
    case "success":
      return "bg-success-muted text-success border-success/40";
    case "error":
      return "bg-error-muted text-error border-error/40";
    default:
      return "bg-primary-muted text-primary border-primary/40";
  }
}
</script>

<template>
  <div class="fixed top-4 right-4 z-[100] flex flex-col gap-2 max-w-sm">
    <Transition
      v-for="toast in toasts"
      :key="toast.id"
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 translate-x-4"
      enter-to-class="opacity-100 translate-x-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-x-0"
      leave-to-class="opacity-0 translate-x-4"
      appear
    >
      <div
        :class="toastClass(toast.type)"
        class="px-4 py-2 text-sm rounded-lg border shadow-lg backdrop-blur-sm"
      >
        {{ toast.message }}
      </div>
    </Transition>
  </div>
</template>
