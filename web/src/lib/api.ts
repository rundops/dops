import type { Catalog, RunbookDetail, ExecutionRecord } from "./types";

const BASE = "/api";

export async function fetchCatalogs(): Promise<Catalog[]> {
  const res = await fetch(`${BASE}/catalogs`);
  if (!res.ok) throw new Error(`Failed to load catalogs: ${res.statusText}`);
  return res.json();
}

export async function fetchRunbook(id: string): Promise<RunbookDetail> {
  const res = await fetch(`${BASE}/runbooks/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error(`Runbook not found: ${id}`);
  return res.json();
}

export async function executeRunbook(
  id: string,
  params: Record<string, string>
): Promise<string> {
  const res = await fetch(`${BASE}/runbooks/${encodeURIComponent(id)}/execute`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ params }),
  });
  if (!res.ok) throw new Error(`Execution failed: ${res.statusText}`);
  const data = await res.json();
  return data.execution_id;
}

export function streamExecution(
  executionId: string,
  onLine: (line: string) => void,
  onDone: (status: string) => void,
  onError: (err: Event) => void
): EventSource {
  const es = new EventSource(`${BASE}/executions/${executionId}/stream`);

  es.onmessage = (event) => {
    onLine(event.data);
  };

  es.addEventListener("done", (event) => {
    onDone((event as MessageEvent).data);
    es.close();
  });

  es.onerror = (event) => {
    onError(event);
    es.close();
  };

  return es;
}

export async function cancelExecution(executionId: string): Promise<void> {
  await fetch(`${BASE}/executions/${executionId}/cancel`, { method: "POST" });
}

export async function fetchHistory(opts?: {
  runbook?: string;
  status?: string;
}): Promise<ExecutionRecord[]> {
  const params = new URLSearchParams();
  if (opts?.runbook) params.set("runbook", opts.runbook);
  if (opts?.status) params.set("status", opts.status);
  const qs = params.toString();
  const res = await fetch(`${BASE}/history${qs ? `?${qs}` : ""}`);
  if (!res.ok) return [];
  return res.json();
}

export async function fetchTheme(): Promise<{
  name: string;
  colors: Record<string, string>;
}> {
  const res = await fetch(`${BASE}/theme`);
  if (!res.ok) return { name: "default", colors: {} };
  return res.json();
}

export async function fetchThemes(): Promise<{
  active: string;
  themes: string[];
}> {
  const res = await fetch(`${BASE}/themes`);
  if (!res.ok) return { active: "", themes: [] };
  return res.json();
}

export async function setTheme(
  name: string
): Promise<{ name: string; colors: Record<string, string> }> {
  const res = await fetch(`${BASE}/theme`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) throw new Error(`Failed to set theme: ${res.statusText}`);
  return res.json();
}
