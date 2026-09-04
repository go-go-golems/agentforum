#!/usr/bin/env bash
set -euo pipefail
review_sources="ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/sources"
mkdir -p "$review_sources"
defuddle parse https://arxiv.org/html/1805.06358v1 --md -o "$review_sources/05-crdt-preguica.md"
defuddle parse https://research.ibm.com/publications/a-relational-model-of-data-for-large-shared-data-banks --md -o "$review_sources/06-codd-publication-abstract.md"
defuddle parse https://web.mit.edu/Saltzer/www/publications/recguides/end-to-end.html --md -o "$review_sources/07-end-to-end-reading-guide.md"
# These original papers are PDF-only; preserve originals and searchable text.
# Defuddle is used above for every HTML source, not for binary PDF parsing.
curl --fail --location --max-time 60 https://lamport.azurewebsites.net/pubs/time-clocks.pdf -o "$review_sources/08-lamport-time-clocks.pdf"
pdftotext -layout "$review_sources/08-lamport-time-clocks.pdf" "$review_sources/08-lamport-time-clocks.txt"
curl --fail --location --max-time 60 https://web.mit.edu/Saltzer/www/publications/endtoend/endtoend.pdf -o "$review_sources/09-end-to-end.pdf"
pdftotext -layout "$review_sources/09-end-to-end.pdf" "$review_sources/09-end-to-end.txt"
