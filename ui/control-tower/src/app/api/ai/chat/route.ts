import { NextRequest, NextResponse } from "next/server";

// Types for AI context
interface AIContext {
  fleetSize?: number;
  driftScore?: number;
  driftedAssets?: number;
  complianceScore?: number;
  drReadiness?: number;
  totalSites?: number;
  criticalAlerts?: number;
}

// System prompt for the AI copilot
function buildSystemPrompt(context?: AIContext): string {
  const contextJson = context ? JSON.stringify(context, null, 2) : "No context available";

  return `You are an AI assistant for QL-RF (Resilience Fabric), an enterprise infrastructure management platform that helps organizations manage golden images, detect configuration drift, ensure compliance, and maintain disaster recovery readiness across multi-cloud environments (AWS, Azure, GCP, VMware, Bare Metal).

CURRENT INFRASTRUCTURE STATE:
${contextJson}

YOUR CAPABILITIES:
1. Analyze drift patterns and identify root causes across multi-cloud environments
2. Assess compliance posture against frameworks (CIS, SLSA, SOC2, HIPAA, PCI-DSS)
3. Recommend specific remediation actions with step-by-step guidance
4. Evaluate DR readiness, RTO/RPO compliance, and site topology
5. Identify optimization opportunities (cost, performance, security)
6. Help users understand their infrastructure health at a glance

RESPONSE GUIDELINES:
- Be concise yet thorough - prioritize actionable insights
- Use technical language appropriate for DevOps and SRE teams
- When discussing issues, always suggest next steps
- Reference specific metrics, assets, sites, or regions when relevant
- Format responses with markdown for readability (headers, bullets, code blocks)
- If you don't have enough context, ask clarifying questions
- For critical issues, emphasize urgency appropriately

AVAILABLE ACTIONS (mention when relevant):
- View affected assets
- Trigger remediation workflows
- Run compliance audits
- Initiate DR drills
- Export reports`;
}

export async function POST(req: NextRequest) {
  try {
    const { message, context, conversationHistory } = await req.json();

    if (!message || typeof message !== "string") {
      return NextResponse.json(
        { error: "Message is required" },
        { status: 400 }
      );
    }

    const endpoint = process.env.AZURE_FOUNDRY_ENDPOINT;
    const apiKey = process.env.AZURE_FOUNDRY_API_KEY;

    if (!endpoint || !apiKey) {
      console.error("Azure Foundry credentials not configured");
      return NextResponse.json(
        { error: "AI service not configured" },
        { status: 500 }
      );
    }

    // Build messages array with conversation history
    const messages = [];

    // Add conversation history if provided
    if (conversationHistory && Array.isArray(conversationHistory)) {
      for (const msg of conversationHistory) {
        messages.push({
          role: msg.role,
          content: msg.content,
        });
      }
    }

    // Add the current user message
    messages.push({
      role: "user",
      content: message,
    });

    // Call Azure Foundry API
    const response = await fetch(endpoint, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "api-key": apiKey,
      },
      body: JSON.stringify({
        model: "claude-sonnet-4-20250514",
        max_tokens: 2048,
        system: buildSystemPrompt(context),
        messages: messages,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error("Azure Foundry API error:", response.status, errorText);
      return NextResponse.json(
        { error: `AI service error: ${response.status}` },
        { status: response.status }
      );
    }

    const data = await response.json();

    // Extract the text content from Claude's response
    const content = data.content?.[0]?.text || "I apologize, but I couldn't generate a response. Please try again.";

    return NextResponse.json({
      content,
      model: data.model,
      usage: data.usage,
    });
  } catch (error) {
    console.error("AI chat error:", error);
    return NextResponse.json(
      { error: "Failed to process AI request" },
      { status: 500 }
    );
  }
}
