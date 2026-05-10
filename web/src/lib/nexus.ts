import { NextResponse } from "next/server";

import type { ErrorResponse, Stats, TraceDetail, TraceSummary } from "@/lib/types";

const DEFAULT_TIMEOUT_MS = 120_000;

export class NexusApiError extends Error {
  status: number;

  constructor(message: string, status = 500) {
    super(message);
    this.name = "NexusApiError";
    this.status = status;
  }
}

export function getNexusApiBaseUrl() {
  const value = process.env.NEXUS_API_BASE_URL;
  if (!value) {
    throw new NexusApiError(
      "NEXUS_API_BASE_URL is not configured for the Next.js app.",
      500,
    );
  }

  return value.endsWith("/") ? value.slice(0, -1) : value;
}

async function readErrorMessage(response: Response) {
  const fallback = `Nexus API request failed with status ${response.status}.`;

  try {
    const data = (await response.json()) as Partial<ErrorResponse>;
    return data.error || fallback;
  } catch {
    return fallback;
  }
}

export async function requestNexus(path: string, init: RequestInit = {}) {
  const response = await fetch(`${getNexusApiBaseUrl()}${path}`, {
    ...init,
    cache: "no-store",
    headers: {
      Accept: "application/json",
      ...(init.headers ?? {}),
    },
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new NexusApiError(await readErrorMessage(response), response.status);
  }

  return response;
}

export async function fetchNexusJson<T>(path: string, init: RequestInit = {}) {
  const response = await requestNexus(path, init);
  return (await response.json()) as T;
}

export function getStats() {
  return fetchNexusJson<Stats>("/stats");
}

export function getTraces(limit = 20) {
  return fetchNexusJson<TraceSummary[]>(`/traces?limit=${limit}`);
}

export function getTrace(id: string) {
  return fetchNexusJson<TraceDetail>(`/traces/${id}`);
}

export function errorResponse(error: unknown) {
  if (error instanceof NexusApiError) {
    return NextResponse.json({ error: error.message }, { status: error.status });
  }

  return NextResponse.json(
    { error: "Unexpected error while contacting Nexus API." },
    { status: 500 },
  );
}
