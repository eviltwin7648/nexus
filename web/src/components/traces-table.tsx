import Link from "next/link";

import type { TraceSummary } from "@/lib/types";
import { cn, formatCurrency, formatDate, formatDuration, formatNumber } from "@/lib/utils";

type TracesTableProps = {
  traces: TraceSummary[];
};

export function TracesTable({ traces }: TracesTableProps) {
  if (traces.length === 0) {
    return (
      <div className="rounded-[28px] border border-dashed border-slate-200 bg-white/80 px-6 py-10 text-sm text-steel shadow-panel">
        No traces yet. Run a few queries through Nexus and they will appear here.
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-[28px] border border-white/70 bg-white/90 shadow-panel">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50/90">
            <tr className="text-left text-xs uppercase tracking-[0.24em] text-steel/70">
              <th className="px-5 py-4 font-medium">Question</th>
              <th className="px-5 py-4 font-medium">Status</th>
              <th className="px-5 py-4 font-medium">Created</th>
              <th className="px-5 py-4 font-medium">Duration</th>
              <th className="px-5 py-4 font-medium">Tokens</th>
              <th className="px-5 py-4 font-medium">Cost</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200/80">
            {traces.map((trace) => (
              <tr key={trace.id} className="align-top">
                <td className="px-5 py-4">
                  <Link
                    href={`/admin/traces/${trace.id}`}
                    className="font-medium text-ink transition hover:text-signal"
                  >
                    <span className="line-clamp-2">{trace.question}</span>
                  </Link>
                  <p className="mt-2 font-mono text-xs text-steel/70">{trace.id}</p>
                </td>
                <td className="px-5 py-4">
                  <span
                    className={cn(
                      "rounded-full px-3 py-1 text-xs font-medium uppercase tracking-[0.18em]",
                      trace.status === "success"
                        ? "bg-green-100 text-pine"
                        : "bg-red-100 text-red-700",
                    )}
                  >
                    {trace.status}
                  </span>
                </td>
                <td className="px-5 py-4 text-steel">{formatDate(trace.created_at)}</td>
                <td className="px-5 py-4 text-steel">{formatDuration(trace.total_ms)}</td>
                <td className="px-5 py-4 text-steel">{formatNumber(trace.total_tokens)}</td>
                <td className="px-5 py-4 text-steel">
                  {formatCurrency(trace.estimated_cost_usd)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
