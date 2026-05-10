import type { Stats } from "@/lib/types";
import { formatCurrency, formatDuration, formatNumber } from "@/lib/utils";

type StatsCardsProps = {
  stats: Stats;
};

const items = (stats: Stats) => [
  { label: "Total queries", value: formatNumber(stats.total_queries) },
  { label: "Successful", value: formatNumber(stats.successful) },
  { label: "Failed", value: formatNumber(stats.failed) },
  { label: "Avg duration", value: formatDuration(Math.round(stats.avg_duration_ms)) },
  { label: "Tokens used", value: formatNumber(stats.total_tokens) },
  { label: "Estimated cost", value: formatCurrency(stats.total_cost_usd) },
];

export function StatsCards({ stats }: StatsCardsProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {items(stats).map((item) => (
        <div
          key={item.label}
          className="rounded-[24px] border border-white/70 bg-white/90 p-5 shadow-panel"
        >
          <p className="font-mono text-xs uppercase tracking-[0.3em] text-steel/70">
            {item.label}
          </p>
          <p className="mt-4 text-3xl font-semibold tracking-tight text-ink">
            {item.value}
          </p>
        </div>
      ))}
    </div>
  );
}
