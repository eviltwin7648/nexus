import type { StepDetail } from "@/lib/types";
import { formatDate, formatNumber } from "@/lib/utils";

type TraceStepListProps = {
  steps: StepDetail[];
};

export function TraceStepList({ steps }: TraceStepListProps) {
  if (steps.length === 0) {
    return (
      <div className="rounded-[24px] border border-dashed border-slate-200 bg-white/80 px-6 py-8 text-sm text-steel">
        This trace completed without recorded tool steps.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {steps.map((step, index) => (
        <section
          key={`${step.tool}-${step.iteration}-${index}`}
          className="rounded-[24px] border border-white/70 bg-white/90 p-5 shadow-panel"
        >
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="font-mono text-xs uppercase tracking-[0.3em] text-steel/70">
                Iteration {step.iteration}
              </p>
              <h3 className="mt-2 text-xl font-semibold tracking-tight text-ink">
                {step.tool}
              </h3>
            </div>
            <div className="grid gap-2 text-sm text-steel sm:grid-cols-3">
              <div>
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-steel/70">
                  Duration
                </p>
                <p className="mt-1 text-ink">{step.duration_ms} ms</p>
              </div>
              <div>
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-steel/70">
                  Output
                </p>
                <p className="mt-1 text-ink">{formatNumber(step.output_len)} chars</p>
              </div>
              <div>
                <p className="font-mono text-[11px] uppercase tracking-[0.22em] text-steel/70">
                  Tokens
                </p>
                <p className="mt-1 text-ink">{formatNumber(step.tokens_used)}</p>
              </div>
            </div>
          </div>
          <div className="mt-5 rounded-[20px] border border-slate-200 bg-slate-950 p-4">
            <p className="mb-3 font-mono text-[11px] uppercase tracking-[0.22em] text-slate-400">
              Input payload · {formatDate(step.created_at)}
            </p>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all text-xs leading-6 text-slate-100">
              {JSON.stringify(step.input, null, 2)}
            </pre>
          </div>
        </section>
      ))}
    </div>
  );
}
