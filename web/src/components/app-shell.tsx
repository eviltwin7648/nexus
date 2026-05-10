import Link from "next/link";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

type AppShellProps = {
  title: string;
  eyebrow: string;
  description: string;
  children: ReactNode;
  active: "workspace" | "admin";
};

const navItems = [
  { href: "/", label: "Workspace", key: "workspace" },
  { href: "/admin", label: "Admin", key: "admin" },
] as const;

export function AppShell({
  title,
  eyebrow,
  description,
  children,
  active,
}: AppShellProps) {
  return (
    <div className="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(14,165,164,0.18),_transparent_28%),radial-gradient(circle_at_top_right,_rgba(249,115,22,0.16),_transparent_24%),linear-gradient(180deg,#f7fbfd_0%,#edf3f8_52%,#f8fbfd_100%)]">
      <div className="pointer-events-none absolute inset-0 bg-grid opacity-50" />
      <div className="relative mx-auto flex min-h-screen w-full max-w-7xl flex-col px-4 py-4 sm:px-6 lg:px-8">
        <header className="mb-6 rounded-[28px] border border-white/70 bg-white/85 p-4 shadow-panel backdrop-blur sm:p-6">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl">
              <p className="mb-3 font-mono text-xs uppercase tracking-[0.35em] text-steel/80">
                {eyebrow}
              </p>
              <h1 className="font-sans text-3xl font-semibold tracking-tight text-ink sm:text-4xl">
                {title}
              </h1>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-steel sm:text-base">
                {description}
              </p>
            </div>
            <nav className="flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50/90 p-1">
              {navItems.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "rounded-full px-4 py-2 text-sm font-medium transition",
                    active === item.key
                      ? "bg-ink text-white"
                      : "text-steel hover:bg-white hover:text-ink",
                  )}
                >
                  {item.label}
                </Link>
              ))}
            </nav>
          </div>
        </header>
        <main className="flex-1">{children}</main>
      </div>
    </div>
  );
}
