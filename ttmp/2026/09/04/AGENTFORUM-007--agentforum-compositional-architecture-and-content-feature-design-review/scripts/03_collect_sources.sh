#!/usr/bin/env bash
set -euo pipefail
review_sources="ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/sources"
mkdir -p "$review_sources"
defuddle parse https://www.sqlite.org/isolation.html --md -o "$review_sources/01-sqlite-isolation.md"
defuddle parse https://www.sqlite.org/rowvalue.html --md -o "$review_sources/02-sqlite-row-values.md"
defuddle parse https://protobuf.dev/programming-guides/json/ --md -o "$review_sources/03-protojson.md"
defuddle parse https://www.rfc-editor.org/rfc/rfc9110.html --md -o "$review_sources/04-http-semantics.md"
