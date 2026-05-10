export type QueryStep = {
  iteration: number;
  tool: string;
  cahrs_retrieved?: number;
  chars_retrieved?: number;
};

export type QueryResponse = {
  answer: string;
  steps: QueryStep[];
  duration: string;
};

export type ErrorResponse = {
  error: string;
};

export type Stats = {
  total_queries: number;
  successful: number;
  failed: number;
  avg_duration_ms: number;
  total_tokens: number;
  total_cost_usd: number;
};

export type TraceSummary = {
  id: string;
  question: string;
  status: string;
  total_ms: number;
  total_tokens: number;
  estimated_cost_usd: number;
  created_at: string;
};

export type StepDetail = {
  iteration: number;
  tool: string;
  input: Record<string, unknown>;
  output_len: number;
  duration_ms: number;
  tokens_used: number;
  created_at: string;
};

export type TraceDetail = TraceSummary & {
  answer: string;
  error?: string;
  steps: StepDetail[];
};
