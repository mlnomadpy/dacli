---
id: d-249-pluggable-optional-semanticscorer-hook-external-dacli-semantic-cmd-backend
kind: note
note_kind: decision
created: 2026-08-04T11:42:06Z
created_by: a-maintainer-kxp25t
about: "[[249]]"
---
# 249: pluggable optional SemanticScorer hook + external $DACLI_SEMANTIC_CMD backend, consulted past the shared-token floor
## Chose
249: pluggable optional SemanticScorer hook + external $DACLI_SEMANTIC_CMD backend, consulted past the shared-token floor
## Rejected
bundle an embedding model / n-gram char similarity into the dedup path so paraphrases are caught with no config
## Because
an in-tree model or heavy heuristic would break the zero-dependency property the task explicitly requires; a nil-by-default hook (tests inject a stub, operators point $DACLI_SEMANTIC_CMD at their own scorer) keeps the default build purely lexical while making paraphrase detection real when opted in. The shared-token floor that guards lexical false positives had to be bypassed for the semantic path, else a zero-word-overlap paraphrase (shared==0) is skipped before the backend is ever asked.
