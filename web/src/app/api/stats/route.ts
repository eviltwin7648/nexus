import { NextResponse } from "next/server";

import { errorResponse, getStats } from "@/lib/nexus";

export const dynamic = "force-dynamic";

export async function GET() {
  try {
    const data = await getStats();
    return NextResponse.json(data);
  } catch (error) {
    return errorResponse(error);
  }
}
