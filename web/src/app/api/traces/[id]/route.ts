import { NextResponse } from "next/server";

import { errorResponse, getTrace } from "@/lib/nexus";

export const dynamic = "force-dynamic";

type RouteContext = {
  params: Promise<{ id: string }>;
};

export async function GET(_: Request, context: RouteContext) {
  try {
    const { id } = await context.params;
    const data = await getTrace(id);
    return NextResponse.json(data);
  } catch (error) {
    return errorResponse(error);
  }
}
