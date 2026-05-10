import { NextRequest, NextResponse } from "next/server";

import { errorResponse, requestNexus } from "@/lib/nexus";

export const dynamic = "force-dynamic";

export async function POST(request: NextRequest) {
  try {
    const body = (await request.json()) as { question?: string };
    if (!body.question?.trim()) {
      return NextResponse.json({ error: "Question is required." }, { status: 400 });
    }

    const response = await requestNexus("/query", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ question: body.question.trim() }),
    });

    const data = await response.json();
    return NextResponse.json(data, { status: 200 });
  } catch (error) {
    return errorResponse(error);
  }
}
