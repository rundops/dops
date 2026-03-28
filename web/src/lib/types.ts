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
