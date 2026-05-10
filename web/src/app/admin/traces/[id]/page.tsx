import Link from "next/link";
import { notFound } from "next/navigation";

import { AppShell } from "@/components/app-shell";
import { MarkdownContent } from "@/components/markdown-content";
import { TraceStepList } from "@/components/trace-step-list";
import { getTrace, NexusApiError } from "@/lib/nexus";
import { formatCurrency, formatDate, formatDuration, formatNumber } from "@/lib/utils";

export const dynamic = "force-dynamic";

type TraceDetailPageProps = {
  params: Promise<{ id: string }>;
};

export default async function TraceDetailPage({ params }: TraceDetailPageProps) {
  const { id } = await params;

  try {
    const trace = await getTrace(id);

    return (
      <AppShell
        active="admin"
        eyebrow="Trace detail"
        title="Inspect a single agent run"
        description="Use the recorded answer, execution status, token usage, and tool payloads to debug the retrieval and reasoning flow."
      >
        <div className="space-y-6">
          <Link href="/admin" className="inline-flex text-sm font-medium text-steel hover:text-ink">
            {"<- Back to admin"}
          </Link>

          <section className="rounded-[28px] border border-white/70 bg-white/90 p-6 shadow-panel">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="max-w-3xl">
                <p className="font-mono text-xs uppercase tracking-[0.32em] text-steel/75">
                  Trace {trace.id}
                </p>
                <h2 className="mt-3 text-2xl font-semibold tracking-tight text-ink">
                  {trace.question}
                </h2>
              </div>
              <span
                className={`rounded-full px-4 py-2 text-xs font-medium uppercase tracking-[0.2em] ${
                  trace.status === "success"
                    ? "bg-green-100 text-pine"
                    : "bg-red-100 text-red-700"
                }`}
              >
                {trace.status}
              </span>
            </div>

            <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Metric label="Created" value={formatDate(trace.created_at)} />
              <Metric label="Duration" value={formatDuration(trace.total_ms)} />
              <Metric label="Tokens" value={formatNumber(trace.total_tokens)} />
              <Metric label="Estimated cost" value={formatCurrency(trace.estimated_cost_usd)} />
            </div>

            <div className="mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(260px,0.8fr)]">
              <article className="rounded-[24px] border border-slate-200 bg-slate-50/70 p-5">
                <p className="font-mono text-xs uppercase tracking-[0.28em] text-steel/70">
                  Answer
                </p>
                <MarkdownContent
                  content={trace.answer || "No answer recorded."}
                  className="mt-4 text-sm text-ink"
                />
              </article>

              <aside className="rounded-[24px] border border-slate-200 bg-slate-950 p-5 text-slate-100">
                <p className="font-mono text-xs uppercase tracking-[0.28em] text-slate-400">
                  Error state
                </p>
                <div className="mt-4 text-sm leading-7 text-slate-100">
                  {trace.error || "No error recorded."}
                </div>
              </aside>
            </div>
          </section>

          <section className="space-y-4">
            <div>
              <p className="font-mono text-xs uppercase tracking-[0.32em] text-steel/75">
                Tool execution timeline
              </p>
              <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink">
                Recorded tool calls
              </h2>
            </div>
            <TraceStepList steps={trace.steps} />
          </section>
        </div>
      </AppShell>
    );
  } catch (error) {
    if (error instanceof NexusApiError && error.status === 404) {
      notFound();
    }

    throw error;
  }
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[20px] border border-slate-200 bg-white px-4 py-4">
      <p className="font-mono text-[11px] uppercase tracking-[0.24em] text-steel/70">{label}</p>
      <p className="mt-2 text-sm font-medium text-ink">{value}</p>
    </div>
  );
}
