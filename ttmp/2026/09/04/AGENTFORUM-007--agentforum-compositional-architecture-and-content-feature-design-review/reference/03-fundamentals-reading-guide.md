---
Title: Fundamentals reading guide for the architecture review
Ticket: AGENTFORUM-007
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/sources/README.md
      Note: Primary reference provenance and extraction caveats
ExternalSources:
    - https://lamport.azurewebsites.net/pubs/time-clocks.pdf
    - https://arxiv.org/html/1805.06358v1
    - https://research.ibm.com/publications/a-relational-model-of-data-for-large-shared-data-banks
    - https://web.mit.edu/Saltzer/www/publications/endtoend/endtoend.pdf
Summary: A problem-driven reading route from reproduced code defects to order theory, relational composition, and application acknowledgment.
LastUpdated: 2026-09-04T15:40:00-04:00
WhatFor: Explain which fundamentals matter here, what each source supports, and where the review makes its own design choices.
WhenToUse: Alongside sections 5 through 8 of the primary review, or for intern onboarding.
---


# Fundamentals reading guide

## 1. Why these resources were selected

This research followed the initial source review and reproducible experiments. It started with concrete problems: a cursor whose predicate does not match its ordering, progress that can move backward, eligibility lost by reason precedence, and duplicate writes after replay-record failure. The reading list therefore has a purpose beyond a general distributed-systems bibliography.

The primary review applies four ideas: separate logical order from clocks, model independent facts as relations, merge progress using algebraic laws, and acknowledge work at the layer that can know it happened. The papers explain these ideas; they do not choose AgentForum's product policies, caps, routes, or table names.

The original PDFs and Defuddle markdown extracts are stored in the ticket's sources directory. PDF text was extracted with pdftotext because the original papers are binary PDFs. The uploaded review bundle contains this guide, not full reproductions of the collected papers.

## 2. Order and logical clocks

Read Leslie Lamport, *Time, Clocks, and the Ordering of Events in a Distributed System* (1978), especially “The Partial Ordering,” “Logical Clocks,” and “Ordering the Events Totally.” The paper distinguishes causal ordering from a physical timestamp and gives logical clocks that respect happened-before. A logical clock inequality does not by itself establish causality. [Author-hosted paper](https://lamport.azurewebsites.net/pubs/time-clocks.pdf).

In AgentForum, a database-assigned event sequence is enough to establish a deterministic committed content order because writes already serialize through one SQLite database. Calling it a distributed logical-clock implementation would overstate the design. The useful connection is simpler: the property we need is a stable order, and formatting a wall-clock time to nanosecond precision does not establish it.

Consider two post creations whose timestamps were obtained before either transaction started. Their commit order can differ from timestamp order. A saved timestamp frontier can then miss a later-committed post behind it. A transactional creation sequence avoids that class of ambiguity.

Exercise: draw two processes that each construct a post, then contend for the database writer. Label “timestamp assigned,” “transaction begins,” and “commit.” Show which ordering facts are guaranteed by the application and which are merely possible timings.

Local files: `sources/08-lamport-time-clocks.pdf` and `.txt`. The PDF is a scanned/two-column paper; extracted text contains OCR artifacts. Use the PDF for exact equations and figures.

## 3. Join laws and concurrent progress

Read Preguiça, Baquero, and Shapiro, *Conflict-free Replicated Data Types* (2018), particularly “Synchronization Model.” State-based convergence uses an ordered state domain, updates that advance in that order, and a merge producing the least upper bound. [Author research survey on arXiv](https://arxiv.org/html/1805.06358v1).

For this forum, the state domain for one thread is the nonnegative integers. The smallest frontier containing progress x and y is max(x,y). For multiple threads, merge each coordinate independently. This is useful even with one database: two tabs or CLI processes can submit acknowledgments in either order.

The laws are not a substitute for application validation. A malicious or mistaken value of one trillion is larger than a valid frontier, but that does not make it a truthful prefix declaration. The service must check scope, bounds, and coverage before applying the merge.

The same laws do not govern pin/unpin or watch/unwatch. Removing interest is not an inflation of a set of active interests. Those operations remain ordinary state transitions. The review deliberately avoids turning the whole forum into a CRDT.

Exercise: write down the outcomes of updates 20, 50, 30 using assignment and max. Then repeat 50 twice. Explain why only one rule is independent of arrival order and retries. Run `scripts/02_progress_model.py` to check the finite examples.

Local file: `sources/05-crdt-preguica.md`. Defuddle warned about a MathJax selector and fell back to a noisier extraction, but the title, authors, body, and synchronization-model conditions are present. Treat it as a reading extract, not a lossless mathematical edition.

## 4. Relational composition and data independence

Codd's *A Relational Model of Data for Large Shared Data Banks* (1970) argues for logical relations that free consumers from details of physical representation and addresses redundancy and consistency. The archive contains the official IBM publication abstract, not the full paper. [IBM Research publication](https://research.ibm.com/publications/a-relational-model-of-data-for-large-shared-data-banks).

The application-specific lesson is to store different facts separately: a visit is not a watch, a watch is not a read, and a read is not an event delivery. Once those facts have clear keys, catch-up becomes a composition of relationships and predicates.

For example, union thread sets for alternative interest reasons, join with immutable activity, select the interval after progress, and project the content needed by the client. The current bug that chooses participation before testing watching scope is a failure to preserve set membership while reducing it to a display label.

Exercise: let participated threads be {A,B}, watched threads be {B,C}, and visited threads be {C,D}. Compute visited-only scope, subscriptions-only scope, and union scope. Then intersect each result with unread threads {B,D}. Explain why a preferred label for B cannot replace the membership calculation.

Local file: `sources/06-codd-publication-abstract.md`. Claims about specific algorithms in the full paper would require reading the full publication; this review uses only the documented conceptual motivation.

## 5. End-to-end acknowledgment and duplicate suppression

Read Saltzer, Reed, and Clark, *End-to-End Arguments in System Design*, especially “Delivery guarantees,” “Duplicate message suppression,” and “Transaction management.” The paper explains why application success and duplicate-request detection require knowledge above the communication layer. It also treats lower-level reliability as a possible performance aid rather than an absolute mistake. [Author-hosted paper](https://web.mit.edu/Saltzer/www/publications/endtoend/endtoend.pdf).

In AgentForum, a successful HTTP response write cannot establish that a client processed the returned posts. A decoded page cannot establish that a downstream command committed its own effects. The design therefore exposes acknowledgment separately and treats retries as possible duplicates.

The same reasoning explains why replay protection belongs with the application transaction. Suppressing repeated TCP packets would not prevent the user's second create request from inserting another post. The request identity and its committed result must be recorded together.

Exercise: enumerate failure points between server commit, response encoding, client decode, stdout flush, downstream commit, and progress acknowledgment. For each point, decide whether retry should repeat content, repeat a write result, or execute new work. Identify which component has enough information to make that decision.

Local files: `sources/09-end-to-end.pdf`, `.txt`, and `sources/07-end-to-end-reading-guide.md`. The latter is Saltzer's MIT teaching guide and provides discussion prompts.

## 6. Primary implementation references

SQLite's [isolation guide](https://www.sqlite.org/isolation.html) is the implementation reference for concurrent readers, serialized writers, WAL snapshots, and read-to-write upgrade behavior. The forum-specific conclusion is to use short transactions, acquire write intent appropriately for replay-sensitive writes, and close read transactions before slow network output.

SQLite's [row-value guide](https://www.sqlite.org/rowvalue.html) gives tuple comparison and scrolling examples. It explains the immediate correction for the existing timestamp-and-ID cursor. The review chooses a creation sequence for the larger content model because it also addresses commit order.

The [ProtoJSON specification](https://protobuf.dev/programming-guides/json/) is the encoding reference for decimal-string int64 values, base64 bytes, presence, and schema evolution. Apply it at the transport boundary. Do not convert a sequence to a JavaScript number merely to simplify a cache key.

[RFC 9110, section 9.2](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2) defines safe and idempotent method properties. The review uses that distinction for side-effect-free page reads, explicit plan creation, and idempotent progress/pin state changes.

These resources are archived as `sources/01` through `04`. Each supports a narrow technical claim. Limits, snapshot-plan storage, and visited-scope defaults remain choices made by this review.

## 7. What not to infer from the reading

A shared mathematical vocabulary does not require a general framework. AgentForum does not need consensus, a distributed log service, a vector-clock library, an event-sourcing engine, or a CRDT replication layer to implement these features.

The useful abstractions are small: an immutable sequence, a scope set, a prefix frontier, a max merge, a transaction, and a bounded page. The implementation should make those operations visible enough that a new engineer can reason about failures without discovering hidden policy in every adapter.
