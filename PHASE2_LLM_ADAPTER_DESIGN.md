# PHASE 2 — LLM ADAPTER DESIGN

> **Status:** Design-Only  
> **Implementation:** Deferred  
> **Last Updated:** 2026-02-08

---

## 1. Purpose & Motivation

### Why Deterministic Rules Are Necessary But Insufficient

The deterministic reasoning engine (Phase 1) provides:

- **Repeatability:** Given identical input, output is always identical.
- **Traceability:** Every hypothesis cites specific evidence.
- **Auditability:** No black-box inference; reasoning is inspectable.
- **Safety:** No hallucination, no invented facts, no surprises.

However, deterministic rules produce *structured data*, not *human understanding*. A hypothesis like:

```
Component: Runtime
Issue: Runtime is only partially ready, indicating dependency or configuration failure
Evidence: ["Runtime default/mydata: Worker 0/2 ready", "Runtime default/mydata: Condition Ready=False, reason=WorkerNotReady"]
```

...is precise, but it requires the operator to:

1. Understand Fluid's architecture
2. Mentally trace the dependency chain
3. Know that "WorkerNotReady" likely means pod scheduling or resource issues
4. Formulate next steps

For experienced SREs, this is fine. For platform teams debugging at 3 AM, or for developers unfamiliar with Fluid internals, the cognitive load is high.

### Why LLMs Excel at Explanation, Not Diagnosis

Large Language Models are effective at:

- Summarizing structured information into prose
- Explaining technical concepts in accessible language
- Generating actionable suggestions from known facts
- Adapting tone and detail level for different audiences

LLMs are **not** effective at:

- Discovering novel correlations in unfamiliar systems
- Avoiding hallucination when grounding is weak
- Maintaining determinism across invocations
- Providing auditable reasoning chains

**Critical distinction:**

| Task | Best Performer |
|------|----------------|
| Diagnosis (what is wrong?) | Deterministic engine |
| Explanation (what does this mean?) | LLM |

### The Core Principle

> **The LLM does not discover issues — it explains existing hypotheses.**

The reasoning output from the deterministic engine is the **source of truth**. The LLM is a **presentation layer** that transforms structured findings into natural language. It has no authority to introduce new facts, modify confidence scores, or override rankings.

---

## 2. Explicit Non-Goals

This design explicitly **excludes** the following capabilities. These are not features "for later" — they are architectural anti-patterns that would compromise the system's integrity.

### The LLM Adapter Will NOT:

| Non-Goal | Rationale |
|----------|-----------|
| **Perform automatic remediation** | Mutation authority must never be delegated to non-deterministic systems. |
| **Execute cluster commands** | No `kubectl`, no API calls, no side effects. |
| **Modify confidence scores** | Confidence is computed deterministically; LLM opinions are not calibrated. |
| **Replace or bypass rules** | Rules are the reasoning engine; the LLM is post-processing only. |
| **Collect logs or resources** | Data collection is upstream; the adapter is read-only. |
| **Make decisions on behalf of operators** | Suggestions only; never prescriptions. |
| **Infer new failure modes** | If the deterministic engine didn't find it, it doesn't exist. |
| **Access external knowledge** | The LLM must be grounded strictly in the provided context. |

### Strong Statements for Reviewers

- **"The LLM is not trusted to infer new facts."**
- **"The LLM has no write access to any system."**
- **"The LLM output is advisory and non-authoritative."**
- **"Disabling the LLM adapter must not degrade diagnostic capability."**

---

## 3. Architectural Placement

### Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         AUTHORITATIVE PATH                          │
│                                                                     │
│   DiagnosticContext                                                 │
│         │                                                           │
│         ▼                                                           │
│   ┌─────────────────────────┐                                       │
│   │ Deterministic Rules     │  ← Pure, stateless, no I/O            │
│   │ Engine (Phase 1)        │                                       │
│   └───────────┬─────────────┘                                       │
│               │                                                     │
│               ▼                                                     │
│   ┌─────────────────────────┐                                       │
│   │ DiagnosisResult         │  ← SOURCE OF TRUTH                    │
│   │ (Hypotheses, Evidence,  │                                       │
│   │  Confidence, Ranking)   │                                       │
│   └───────────┬─────────────┘                                       │
│               │                                                     │
└───────────────┼─────────────────────────────────────────────────────┘
                │
                │  (optional, one-way)
                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         ADVISORY PATH                               │
│                                                                     │
│   ┌─────────────────────────┐                                       │
│   │ LLM Adapter             │  ← Read-only consumer                 │
│   │ (Optional)              │                                       │
│   └───────────┬─────────────┘                                       │
│               │                                                     │
│               ▼                                                     │
│   ┌─────────────────────────┐                                       │
│   │ Explanation             │  ← Human-readable, non-authoritative  │
│   │ (Natural Language)      │                                       │
│   └─────────────────────────┘                                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Properties

1. **One-way data flow:** Information flows from the engine to the adapter, never back.
2. **No feedback loop:** The LLM cannot influence rule execution or scoring.
3. **Optional by default:** The system is fully functional without the adapter.
4. **Additive only:** The adapter adds explanation; it cannot subtract or modify findings.

---

## 4. Adapter Interface (Design Only)

### Conceptual Interface

The adapter exposes a single capability:

```
Explain(result: DiagnosisResult, context: DiagnosticContext) → Explanation
```

### Input Contracts

| Input | Type | Access |
|-------|------|--------|
| `result` | `DiagnosisResult` | Read-only, immutable |
| `context` | `DiagnosticContext` | Read-only, immutable (may be filtered) |

The adapter **receives** these inputs; it does not fetch or construct them.

### Output Contract

```
Explanation:
  Summary: string           # High-level, human-readable summary
  HypothesisDetails: []     # Per-hypothesis explanations
    - Rank: int
    - PlainLanguage: string # What this means in plain English
    - WhyThisMatters: string
    - SuggestedActions: []string
    - EvidenceCited: []string  # Must be subset of Hypothesis.Evidence
  Caveats: []string         # Disclaimers about limitations
  GeneratedBy: string       # Model identifier for auditability
```

### Critical Properties

- **Output is non-authoritative:** The `Explanation` is for human consumption only. It is never fed back into the system.
- **Evidence subset enforcement:** `EvidenceCited` must be a strict subset of `Hypothesis.Evidence`. No new evidence can be introduced.
- **No confidence manipulation:** The explanation does not include any confidence reinterpretation.

---

## 5. Prompt Boundary Design

### What the LLM Is Allowed to See

| Data | Allowed | Justification |
|------|---------|---------------|
| Hypotheses (ranked) | ✅ Yes | Core data to explain |
| Evidence strings | ✅ Yes | Required for grounding |
| Component names | ✅ Yes | Context for explanation |
| Suggestions from rules | ✅ Yes | Base for elaboration |
| Cluster version | ✅ Yes | Contextual |
| Namespace | ✅ Yes | Contextual |
| Truncated logs (≤500 chars per source) | ✅ Yes | Additional context, bounded |

### What the LLM Must Never See

| Data | Reason |
|------|--------|
| Full, untruncated logs | Token limit, potential secrets |
| Raw Kubernetes manifests | May contain secrets, excessive size |
| Service account tokens | Security |
| Network policies | Security |
| Environment variables | Potential secrets |
| Any data not in `DiagnosticContext` | Grounding violation |

### Log Truncation Strategy

Logs are pre-processed before inclusion:

1. **Length limit:** Maximum 500 characters per log source.
2. **Recency bias:** Prefer last N lines over first N lines.
3. **Error highlighting:** Prioritize lines containing `ERROR`, `FATAL`, `panic`.
4. **Redaction:** Remove patterns matching secrets (base64 blobs, tokens, passwords).

### Sample Prompt Skeleton

```
SYSTEM:
You are an assistant that explains Fluid diagnostic results.
You do NOT diagnose issues — that has already been done.
Your job is to explain the provided hypotheses in plain language.

STRICT INSTRUCTIONS:
- You may ONLY reference evidence that is explicitly provided below.
- You may NOT introduce new failure causes or hypotheses.
- You may NOT claim certainty — all findings are hypotheses.
- You may NOT suggest commands that mutate the cluster.
- You MUST cite evidence IDs when making claims.
- If you are unsure, say "Based on the available evidence..."

CONTEXT:
Cluster Version: v1.28.0
Namespace: default
Collection Time: 2026-02-08T04:35:00Z

HYPOTHESES (in ranked order):

[1] Component: Fuse
    Issue: Fuse pod cannot be scheduled due to node taints or missing tolerations
    Confidence: 0.8
    Evidence:
      - E1: Pod default/mydata-fuse-abc123: PodScheduled=False, reason=Unschedulable
      - E2: Event: FailedScheduling - 0/1 nodes are available: 1 node(s) had taints
    Suggestion: Check node taints and ensure Fuse pods have appropriate tolerations.

[2] Component: Runtime
    Issue: Runtime is only partially ready
    Confidence: 0.6
    Evidence:
      - E3: Runtime default/mydata: Worker 0/2 ready
    Suggestion: Check runtime pod logs for errors.

TASK:
For each hypothesis, provide:
1. A plain-language explanation of what this means
2. Why this matters for the application
3. Concrete next steps (read-only commands only, e.g., kubectl get, kubectl describe)
4. Which evidence supports your explanation (cite E1, E2, etc.)

Begin your response with a brief overall summary.
```

---

## 6. Grounding & Hallucination Prevention

### The Hallucination Problem

LLMs can generate plausible-sounding but factually incorrect statements. In a diagnostic context, this is dangerous:

- "The node is out of disk space" (when no evidence suggests this)
- "The application is crashing due to OOMKill" (when no OOMKill events exist)
- "You should delete the pod and recreate it" (unsolicited mutation advice)

### Grounding Mechanisms

#### 6.1 Evidence ID Referencing

Every claim in the explanation must cite evidence:

```
✅ ALLOWED:
"The Fuse pod is unschedulable (E1) because the node has taints 
that the pod does not tolerate (E2)."

❌ FORBIDDEN:
"The Fuse pod is unschedulable due to network policy restrictions."
(No evidence E-ID supports this claim)
```

#### 6.2 Fixed Response Structure

The LLM output must conform to a template:

```
Summary: <2-3 sentences>

Hypothesis 1 Explanation:
- Plain Language: <explanation citing E-IDs>
- Why It Matters: <impact statement>
- Suggested Actions: <list of read-only commands>
- Evidence Used: [E1, E2]

Hypothesis 2 Explanation:
...

Caveats:
- This analysis is based on a point-in-time snapshot.
- Additional issues may exist that were not detected.
```

#### 6.3 Rejection of Unsupported Claims

Conceptually, a post-processing validator would:

1. Parse the LLM response.
2. Extract all E-ID references.
3. Verify each E-ID exists in the provided evidence.
4. Flag or strip claims that reference non-existent evidence.
5. Reject responses that introduce new hypotheses or components.

**Design note:** This validator would be implemented in a future phase. The design here ensures it is architecturally possible.

#### 6.4 Ranked Hypotheses Only

The LLM must explain hypotheses in the order provided. It cannot:

- Reorder hypotheses
- Combine multiple hypotheses into one
- Dismiss low-ranked hypotheses as "unlikely"
- Promote its own preferred explanation

---

## 7. Confidence & Responsibility Model

### Confidence Ownership

| Aspect | Owner |
|--------|-------|
| Confidence score calculation | Deterministic engine (Phase 1) |
| Confidence score modification | **Forbidden** |
| Confidence interpretation | LLM (advisory only) |

### What the LLM May Say About Confidence

```
✅ ALLOWED:
"This hypothesis has a confidence score of 0.8, which indicates 
strong signal correlation between the event and pod status."

❌ FORBIDDEN:
"I believe this is actually 0.95 confidence because..."
"This seems more like a 0.4 to me..."
"The confidence score is misleading; the real cause is..."
```

### Responsibility Chain

```
┌─────────────────────────────────────────────────────────────────┐
│ DETERMINISTIC ENGINE                                            │
│ Responsibilities:                                               │
│ - Issue detection                                               │
│ - Evidence collection                                           │
│ - Confidence scoring                                            │
│ - Hypothesis ranking                                            │
│ Accountable for: Correctness of diagnosis                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ LLM ADAPTER                                                     │
│ Responsibilities:                                               │
│ - Summarization                                                 │
│ - Plain-language explanation                                    │
│ - Suggested actions (read-only)                                 │
│ Accountable for: Clarity of explanation (NOT diagnosis)         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ HUMAN OPERATOR                                                  │
│ Responsibilities:                                               │
│ - Final decision making                                         │
│ - Action execution                                              │
│ - Judgment on whether to proceed                                │
│ Accountable for: Cluster mutations                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Enterprise & CNCF Safety Considerations

### 8.1 Offline-First Default

The system **must** function without any LLM:

- Default mode: LLM adapter disabled.
- Explicit opt-in required for LLM features.
- No degradation when LLM is unavailable.
- All capabilities except natural-language explanation remain intact.

### 8.2 Pluggable Backends

The adapter design supports multiple backends:

| Backend Type | Example | Notes |
|--------------|---------|-------|
| None (disabled) | — | Default, always available |
| Local model | Ollama, llama.cpp | Air-gapped environments |
| Self-hosted | vLLM, text-generation-inference | Enterprise deployment |
| Cloud API | OpenAI, Anthropic, Vertex | Opt-in, with data governance |

**No hard dependency on any SaaS provider.**

### 8.3 Data Governance

Before invoking any LLM:

1. **Data classification:** Only pre-approved data categories are transmitted.
2. **Log redaction:** Secrets, tokens, and PII are stripped.
3. **Audit logging:** Every LLM invocation is logged with:
   - Timestamp
   - Input hash (not full content)
   - Model identifier
   - User/service account
4. **Retention policy:** LLM requests are not cached beyond the session.

### 8.4 Auditable Reasoning Chain

For compliance and debugging:

```
DiagnosticBundle:
  Input: DiagnosticContext (hash: abc123)
  Engine: rule-based v1.0.0
  Result: DiagnosisResult (hash: def456)
  Adapter:
    Enabled: true
    Backend: ollama/mistral-7b
    Prompt: (hash: ghi789)
    Response: (hash: jkl012)
  Explanation: <stored separately if needed>
```

Every step is traceable. If a user asks "why did it say X?", the chain is reconstructable.

### 8.5 CNCF Alignment

This design aligns with CNCF principles:

- **Vendor-neutral:** No lock-in to specific AI providers.
- **Observable:** Reasoning is inspectable and auditable.
- **Secure by default:** Minimal data exposure, opt-in features.
- **Composable:** Works with existing Fluid tooling.

---

## 9. Why This Is Design-Only

### Justification for Deferring Implementation

This phase produces **architecture, not code** for deliberate reasons:

#### 9.1 Premature AI Integration Is Dangerous

Many projects rush to add "AI features" without:

- Understanding failure modes
- Defining trust boundaries
- Establishing grounding requirements
- Planning for degradation

The result is systems where:

- Users don't know which outputs are trustworthy
- Hallucinations cause real-world incidents
- "AI said so" becomes an excuse for not understanding the system
- Debugging becomes impossible

**This design ensures we add AI correctly, not quickly.**

#### 9.2 The Deterministic Engine Must Prove Itself First

Phase 1 established:

- The reasoning is sound
- The evidence is traceable
- The output is useful without explanation

Only after operators trust the deterministic output should we layer explanation on top. Adding AI before this trust exists would confuse the value proposition.

#### 9.3 Interface Stability

The adapter interface depends on:

- `DiagnosisResult` structure (stable)
- `DiagnosticContext` structure (stable)
- Consumer expectations (evolving)

By designing now but implementing later, we:

- Validate the interface with stakeholders
- Collect feedback on explanation needs
- Avoid rework from premature commitments

#### 9.4 Enterprise Readiness

Organizations adopting this tool need:

- Security review of data flows
- Legal review of AI provider agreements
- Procurement of approved models/providers
- Training for operators

A design document enables these processes to proceed in parallel with engineering work.

### What This Design Enables

A future implementer can:

1. Read this document and understand the boundaries.
2. Implement the adapter with confidence about what is allowed.
3. Choose an appropriate backend without architectural changes.
4. Integrate with enterprise governance workflows.
5. Extend the system safely without breaking determinism.

---

## Appendix A: Terminology

| Term | Definition |
|------|------------|
| **Authoritative** | Data or output that is the source of truth for downstream consumers. |
| **Advisory** | Data or output that is informational only; not to be trusted for automated decisions. |
| **Grounding** | Constraining LLM output to reference only provided evidence. |
| **Hallucination** | LLM-generated content that is factually incorrect or not supported by input. |
| **Evidence ID** | A unique identifier for a piece of evidence (e.g., E1, E2) used for citation. |

---

## Appendix B: Review Checklist

For maintainers reviewing this design:

- [ ] Does the adapter have any write access? (Should be: No)
- [ ] Can the LLM modify confidence scores? (Should be: No)
- [ ] Is there a feedback loop to the rules engine? (Should be: No)
- [ ] Is the system functional without the adapter? (Should be: Yes)
- [ ] Are secrets protected from LLM exposure? (Should be: Yes)
- [ ] Is the reasoning chain auditable? (Should be: Yes)
- [ ] Is there hard dependency on a SaaS provider? (Should be: No)

---

## Appendix C: Related Documents

- `PHASE0_DESIGN.md` — Problem statement and core contracts
- `README.md` — Usage and integration guide

---

*This document was authored as part of fluid-ai-diagnoser Phase 2 (Design Only).*
