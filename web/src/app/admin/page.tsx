import { AppShell } from "@/components/app-shell";
import { StatsCards } from "@/components/stats-cards";
import { TracesTable } from "@/components/traces-table";
import { getStats, getTraces, NexusApiError } from "@/lib/nexus";

export const dynamic = "force-dynamic";

export default async function AdminPage() {
  try {
    const [stats, traces] = await Promise.all([getStats(), getTraces(30)]);

    return (
      <AppShell
        active="admin"
        eyebrow="Operations view"
        title="Trace health, cost, and recent query traffic"
        description="The admin surface reads the existing Nexus observability endpoints and turns them into a fast operator console for recent traces and aggregate usage."
      >
        <div className="space-y-6">
          <StatsCards stats={stats} />
          <section className="space-y-4">
            <div>
              <p className="font-mono text-xs uppercase tracking-[0.32em] text-steel/75">
                Recent traces
              </p>
              <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink">
                Latest 30 query runs
              </h2>
            </div>
            <TracesTable traces={traces} />
          </section>
        </div>
      </AppShell>
    );
  } catch (error) {
    const message =
      error instanceof NexusApiError
        ? error.message
        : "Unable to load observability data from the Nexus API.";

    return (
      <AppShell
        active="admin"
        eyebrow="Operations view"
        title="Trace health, cost, and recent query traffic"
        description="The admin surface reads the existing Nexus observability endpoints and turns them into a fast operator console for recent traces and aggregate usage."
      >
        <div className="rounded-[28px] border border-red-200 bg-red-50 p-6 text-red-700 shadow-panel">
          {message}
        </div>
      </AppShell>
    );
  }
}
