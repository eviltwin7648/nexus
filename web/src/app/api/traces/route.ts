import { NextRequest, NextResponse } from "next/server";

import { errorResponse, getTraces } from "@/lib/nexus";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  try {
    const limitValue = request.nextUrl.searchParams.get("limit");
    const limit = limitValue ? Number.parseInt(limitValue, 10) : 20;
    const data = await getTraces(Number.isFinite(limit) ? limit : 20);
    return NextResponse.json(data);
  } catch (error) {
    return errorResponse(error);
  }
}
