export interface Catalog {
  name: string;
  display_name?: string;
  path: string;
  active: boolean;
  runbooks: RunbookSummary[];
}

export interface RunbookSummary {
  id: string;
  name: string;
  description: string;
  risk_level: string;
  param_count: number;
}

export interface RunbookDetail {
  id: string;
  name: string;
  aliases?: string[];
  description: string;
  version: string;
  risk_level: string;
  script: string;
  parameters: Parameter[];
  saved_values?: Record<string, string>;
}

export interface ExecutionRecord {
  id: string;
  runbook_id: string;
  runbook_name: string;
  catalog_name: string;
  parameters?: Record<string, string>;
  status: "running" | "success" | "failed" | "cancelled";
  exit_code: number;
  start_time: string;
  end_time?: string;
  duration?: string;
  output_lines: number;
  output_summary?: string;
  log_path?: string;
  interface: "tui" | "cli" | "web" | "mcp";
}

export interface Parameter {
  name: string;
  type: string;
  required: boolean;
  scope: string;
  secret: boolean;
  default?: unknown;
  description: string;
  options?: string[];
}
