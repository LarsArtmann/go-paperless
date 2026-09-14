# ADR 0001: API-version policy beyond v10

- Status: accepted
- Date: 2026-09-14
- Deciders: Lars Artmann

## Context

Paperless-ngx negotiates API versions via the `Accept:
application/json; version=10` header. The v10 endpoint shapes differ from
older deployments: document listings paginate (`{"results": [...]}` vs.
bare arrays), checksums moved into `versions[]` on 3.x, and task endpoints
gained envelopes. The SDK pins one version (`apiVersion = "10"`) but
decodes tolerantly.

Two futures are possible:

1. **Paperless-ngx ships v11+** and deprecates v10.
2. The project's API stays effectively frozen (it has been for years) and
   self-hosted deployments trail far behind releases anyway.

Consumers (InboxClean, bank-sync) run against long-lived self-hosted
servers. They need the SDK to keep working across server upgrades without
SDK majors.

## Decision

- **Pin one target version** (`apiVersion`) per SDK major. The pinned
  version is what we test against; it changes only in a deliberate
  release.
- **Read, tolerate, never emulate.** We probe the server's negotiated
  version (`ProbeCapabilities.AcceptAPIVersion`) and decode tolerantly
  (flat checksums, bare-array task fallbacks), but we do not emulate every
  historical version's quirks in request construction. Requests always
  speak the pinned version.
- **Detect drift, don't guess.** When the server negotiates a version
  different from the pinned one, that observation is data (exposed via
  `Capabilities`), not an automatic behavior switch.
- **Major-version SDK bumps track major-version API moves.** If v11
  removes v10 shapes we depend on, the fix is a new SDK major (or a
  documented capability gate), not silent runtime adaptation.

## Consequences

- New server versions do not force an SDK release: tolerant decoding and
  capability probing absorb additive changes.
- Truly breaking server changes surface as decode/capability errors with
  `paperless.*` codes — diagnosable, not silently wrong.
- Consumers get stability: no behavior flips based on what the server
  negotiates.
- Cost: some future server features (new fields, new endpoints) need
  explicit SDK work rather than arriving automatically.
