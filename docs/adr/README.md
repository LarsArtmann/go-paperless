# Architecture Decision Records

Every decision that shapes this SDK's public contract or internal policy
gets a numbered ADR here. Statuses: `proposed` → `accepted` (or
`superseded by NNNN`); accepted records are never edited into something
they did not say — supersede instead.

## Index

| ADR  | Title                                            | Status   |
| ---- | ------------------------------------------------ | -------- |
| 0001 | [API-version policy beyond v10](0001-api-version-policy.md) | accepted |
| 0002 | [Error model](0002-error-model.md)               | accepted |
| 0003 | [Streaming upload is not promised](0003-streaming-upload.md) | accepted |

## Process

1. Copy [`template.md`](template.md) to `NNNN-short-title.md` (next free
   number, kebab-case title).
2. Fill every section; if Context cannot name a concrete force (a bug, a
   consumer, a server behavior), the decision is not ready to record.
3. Cross-link from the affected surface (README section, FEATURES row, or
   code comment) so the record is discoverable where the decision applies.

Related catalogs: error codes live in
[docs/ERROR_CODES.md](../ERROR_CODES.md); the error-model ADR (0002) is
enforced by the erraudit gate described in `AGENTS.md`.
