import { AppShell } from "@/components/app-shell";
import { QueryWorkbench } from "@/components/query-workbench";

export default function HomePage() {
  return (
    <AppShell
      active="workspace"
      eyebrow="Nexus knowledge engine"
      title="Ask the codebase without exposing the backend directly"
      description="This workspace is the human-facing layer for Nexus. It forwards browser requests through Next.js route handlers, submits them to the existing Go API, and presents answers with the agent’s execution footprint."
    >
      <QueryWorkbench />
    </AppShell>
  );
}
