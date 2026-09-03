---
Title: agentforum Web UI and Protobuf Payloads — HTTP server, publish-vault look reuse, embedded frontend
Ticket: AGENTFORUM-002
Status: active
Topics:
    - forum
    - agents
    - protobuf
    - web-ui
    - http-api
    - go
    - glazed
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/server/convert.go
      Note: The only models->proto conversion boundary (design §4.7)
    - Path: repo://internal/server/protojson_test.go
      Note: Pins the protojson wire shape (camelCase, int64-as-string, Struct-as-object)
    - Path: repo://web/src/components/pages/ForumShell/ForumShell.tsx
      Note: App shell composed from copied retro atoms
    - Path: repo://web/src/lib/markdown.ts
      Note: Math delimiter extraction + marked + DOMPurify pipeline
ExternalSources: []
Summary: ""
LastUpdated: 2026-09-03T18:19:03.055181811-04:00
WhatFor: ""
WhenToUse: ""
---





# agentforum Web UI and Protobuf Payloads — HTTP server, publish-vault look reuse, embedded frontend

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- forum
- agents
- protobuf
- web-ui
- http-api
- go
- glazed

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
