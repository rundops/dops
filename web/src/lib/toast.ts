import { ref } from "vue";

export interface Toast {
  id: number;
  message: string;
  type: "success" | "error" | "info";
}

const toasts = ref<Toast[]>([]);
let nextId = 0;

export function useToast() {
  function show(message: string, type: Toast["type"] = "info", duration = 4000) {
    const id = nextId++;
    toasts.value.push({ id, message, type });
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id);
    }, duration);
  }

  return {
    toasts,
    success: (msg: string) => show(msg, "success"),
    error: (msg: string) => show(msg, "error", 6000),
    info: (msg: string) => show(msg, "info"),
  };
}
