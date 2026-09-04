# Source archive

Collected 2026-09-04 after the source review identified the relevant correctness questions. Run scripts 03 and 04 from the repository root to repeat collection. Web pages were extracted with Defuddle; binary PDFs were downloaded from author-hosted URLs and converted to searchable text with pdftotext.

| Local file | Source | Use and qualification |
|---|---|---|
| 01-sqlite-isolation.md | https://www.sqlite.org/isolation.html | Transactions, WAL snapshots, write intent |
| 02-sqlite-row-values.md | https://www.sqlite.org/rowvalue.html | Composite cursor comparisons |
| 03-protojson.md | https://protobuf.dev/programming-guides/json/ | Bytes, int64, field presence, schema evolution |
| 04-http-semantics.md | https://www.rfc-editor.org/rfc/rfc9110.html | Safe/idempotent methods; long full-spec extract |
| 05-crdt-preguica.md | https://arxiv.org/html/1805.06358v1 | Preguiça, Baquero, Shapiro (2018); complete body with extraction noise |
| 06-codd-publication-abstract.md | https://research.ibm.com/publications/a-relational-model-of-data-for-large-shared-data-banks | Codd (1970), official abstract only |
| 07-end-to-end-reading-guide.md | https://web.mit.edu/Saltzer/www/publications/recguides/end-to-end.html | Author's teaching guide |
| 08-lamport-time-clocks.pdf / .txt | https://lamport.azurewebsites.net/pubs/time-clocks.pdf | Lamport (1978); original PDF plus imperfect OCR/text order |
| 09-end-to-end.pdf / .txt | https://web.mit.edu/Saltzer/www/publications/endtoend/endtoend.pdf | Saltzer, Reed, Clark; original author-hosted paper and text |

The RFC extractor printed a stylesheet parsing warning but wrote a substantial markdown document. The arXiv extractor printed a MathJax selector warning but retained the article body, including the join-semilattice conditions. These warnings are recorded rather than silently describing the extracts as perfect. Consult the source or original PDF for exact mathematical typography.

The final reMarkable bundle includes original review prose and the annotated reading guide. It does not concatenate the full archived third-party texts.
