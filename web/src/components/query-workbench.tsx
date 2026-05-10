"use client";

import { useEffect, useState, useTransition } from "react";

import { MarkdownContent } from "@/components/markdown-content";
import type { ErrorResponse, QueryResponse } from "@/lib/types";
import { formatDate } from "@/lib/utils";

type HistoryEntry = {
  id: string;
  question: string;
  answer: string;
  duration: string;
  createdAt: string;
};

const HISTORY_KEY = "nexus-query-history";

export function QueryWorkbench() {
  const [question, setQuestion] = useState("");
  const [result, setResult] = useState<QueryResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [, startTransition] = useTransition();

  useEffect(() => {
    const raw = window.sessionStorage.getItem(HISTORY_KEY);
    if (!raw) {
      return;
    }

    try {
      const parsed = JSON.parse(raw) as HistoryEntry[];
      setHistory(parsed);
    } catch {
      window.sessionStorage.removeItem(HISTORY_KEY);
    }
  }, []);

  useEffect(() => {
    window.sessionStorage.setItem(HISTORY_KEY, JSON.stringify(history));
  }, [history]);

  const submit = () => {
    const trimmed = question.trim();
    if (!trimmed || isRunning) {
      return;
    }

    setError(null);
    setIsRunning(true);

    void (async () => {
      try {
        const response = await fetch("/api/query", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ question: trimmed }),
        });

        if (!response.ok) {
          const data = (await response.json()) as ErrorResponse;
          throw new Error(data.error || "Query failed.");
        }

        const data = (await response.json()) as QueryResponse;
        startTransition(() => {
          setResult(data);
          setHistory((current) =>
            [
              {
                id: crypto.randomUUID(),
                question: trimmed,
                answer: data.answer,
                duration: data.duration,
                createdAt: new Date().toISOString(),
              },
              ...current,
            ].slice(0, 6),
          );
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Query failed.");
      } finally {
        setIsRunning(false);
      }
    })();
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1.55fr)_minmax(320px,0.9fr)]">
      <section className="rounded-[28px] border border-white/70 bg-white/90 p-4 shadow-panel backdrop-blur sm:p-6">
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="font-mono text-xs uppercase tracking-[0.32em] text-steel/75">
                Ask Nexus
              </p>
              <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink">
                Query your indexed repos and notes
              </h2>
            </div>
            <div className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs text-steel">
              Tool-calling RAG
            </div>
          </div>

          <label className="block">
            <span className="sr-only">Question</span>
            <textarea
              value={question}
              onChange={(event) => setQuestion(event.target.value)}
              rows={8}
              placeholder="Ask about architecture decisions, code paths, operational behavior, or note content."
              className="min-h-[220px] w-full rounded-[24px] border border-slate-200 bg-slate-50/80 px-5 py-4 text-base text-ink outline-none transition placeholder:text-slate-400 focus:border-signal focus:bg-white"
            />
          </label>

          <div className="flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={submit}
              disabled={isRunning || question.trim().length === 0}
              className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white transition hover:bg-steel disabled:cursor-not-allowed disabled:bg-slate-300"
            >
              {isRunning ? "Running query..." : "Run query"}
            </button>
            <button
              type="button"
              onClick={() => {
                setQuestion("");
                setResult(null);
                setError(null);
              }}
              className="rounded-full border border-slate-200 bg-white px-5 py-3 text-sm font-medium text-steel transition hover:border-slate-300 hover:text-ink"
            >
              Clear
            </button>
          </div>

          {error ? (
            <div className="rounded-[22px] border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          <div className="rounded-[24px] border border-slate-200 bg-[linear-gradient(180deg,#ffffff_0%,#f6fafc_100%)] p-5">
            {isRunning && !result ? (
              <div className="space-y-3">
                <div className="h-4 w-36 animate-pulse rounded-full bg-slate-200" />
                <div className="h-4 w-full animate-pulse rounded-full bg-slate-200" />
                <div className="h-4 w-5/6 animate-pulse rounded-full bg-slate-200" />
                <div className="h-4 w-2/3 animate-pulse rounded-full bg-slate-200" />
              </div>
            ) : result ? (
              <div className="space-y-5">
                <div className="flex flex-wrap items-center gap-3">
                  <span className="rounded-full bg-signal/10 px-3 py-1 font-mono text-xs uppercase tracking-[0.24em] text-signal">
                    Answer
                  </span>
                  <span className="text-sm text-steel">Completed in {result.duration}</span>
                </div>
                <MarkdownContent content={result.answer} className="text-sm text-ink sm:text-[15px]" />
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                  {result.steps.map((step, index) => (
                    <div
                      key={`${step.tool}-${step.iteration}-${index}`}
                      className="rounded-[20px] border border-slate-200 bg-white p-4"
                    >
                      <p className="font-mono text-xs uppercase tracking-[0.26em] text-steel/70">
                        Iteration {step.iteration}
                      </p>
                      <h3 className="mt-2 text-sm font-semibold text-ink">{step.tool}</h3>
                      <p className="mt-2 text-sm text-steel">
                        Retrieved{" "}
                        {step.cahrs_retrieved ?? step.chars_retrieved ?? 0} characters
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="flex min-h-[240px] items-center justify-center rounded-[20px] border border-dashed border-slate-200 bg-white/60 px-6 text-center text-sm leading-6 text-steel">
                Submit a question to see Nexus answer with retrieved context and tool trace hints.
              </div>
            )}
          </div>
        </div>
      </section>

      <aside className="space-y-6">
        <section className="rounded-[28px] border border-white/70 bg-white/90 p-5 shadow-panel backdrop-blur">
          <p className="font-mono text-xs uppercase tracking-[0.32em] text-steel/75">
            Session history
          </p>
          <div className="mt-4 space-y-3">
            {history.length === 0 ? (
              <div className="rounded-[20px] border border-dashed border-slate-200 bg-slate-50/80 px-4 py-6 text-sm text-steel">
                Query history stays in this browser session only.
              </div>
            ) : (
              history.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  onClick={() => {
                    setQuestion(entry.question);
                    setResult({
                      answer: entry.answer,
                      duration: entry.duration,
                      steps: [],
                    });
                    setError(null);
                  }}
                  className="w-full rounded-[20px] border border-slate-200 bg-slate-50/80 px-4 py-4 text-left transition hover:border-signal/40 hover:bg-white"
                >
                  <p className="line-clamp-2 text-sm font-medium leading-6 text-ink">
                    {entry.question}
                  </p>
                  <div className="mt-3 flex items-center justify-between gap-3 text-xs text-steel">
                    <span>{entry.duration}</span>
                    <span>{formatDate(entry.createdAt)}</span>
                  </div>
                </button>
              ))
            )}
          </div>
        </section>

        <section className="rounded-[28px] border border-ink/90 bg-ink p-5 text-white shadow-panel">
          <p className="font-mono text-xs uppercase tracking-[0.32em] text-white/70">
            Integration note
          </p>
          <p className="mt-3 text-sm leading-6 text-white/82">
            Browser requests terminate in Next.js route handlers, which proxy to the Go API.
            That keeps the backend unchanged and avoids browser CORS issues.
          </p>
        </section>
      </aside>
    </div>
  );
}
