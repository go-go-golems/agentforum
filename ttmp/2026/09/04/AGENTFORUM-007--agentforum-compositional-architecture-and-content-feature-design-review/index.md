---
Title: AgentForum compositional architecture and content feature design review
Ticket: AGENTFORUM-007
Status: review
Topics:
    - backend
    - frontend
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/models/models.go
      Note: Domain model under review
    - Path: repo://internal/store/migrations/0001_init.sql
      Note: Current data constraints
    - Path: repo://web/src/hooks/useEventStream.ts
      Note: Transport and application progress boundaries
ExternalSources: []
Summary: Architecture review of AGENTFORUM-001 through 006, with reproducible defects and a fundamentals-based implementation design.
LastUpdated: 2026-09-04T15:25:23.730721615-04:00
WhatFor: Make the content-feature design coherent and reviewable before implementation.
WhenToUse: Before starting AGENTFORUM-006 or revisiting the remote service boundary.
---


# AgentForum compositional architecture and content feature design review

## Overview

Review the existing tickets, diaries, and application at commit `90a343c`, then propose a compositional architecture for AGENTFORUM-006. The design separates immutable content order, mutable pin display, explicit interest, monotonic progress, and delivery acknowledgment.

This ticket contains review artifacts and experiments only. Production implementation remains future work. Existing Go internal tests, web type checks, and all 16 web tests passed; six ticket-local Go probes reproduced missing boundary guarantees.

## Key Links

- [Primary architecture and implementation review](design-doc/01-fundamentals-based-architecture-and-agentforum-006-implementation-review.md)
- [Experiment results and baseline validation](reference/02-experiment-results.md)
- [Annotated fundamentals reading guide](reference/03-fundamentals-reading-guide.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Collected sources and provenance](sources/README.md)
- [Reproducible scripts](scripts/)

## Status

Current status: **review** — analysis and delivery complete; design decisions remain proposed.

Delivered to reMarkable at `/ai/2026/09/04/AGENTFORUM-007` as **AGENTFORUM-007 Architecture and Content Design Review**. The upload command reported success after a clean dry run. Docmgr doctor passed. The PDF contains the primary review, fundamentals guide, experiment record, and investigation diary Steps 1–2; the local diary additionally records delivery in Step 3.

## Topics

- backend
- frontend
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- sources/ - Primary references, Defuddle extracts, and original research PDFs
- archive/ - Deprecated or reference-only artifacts
