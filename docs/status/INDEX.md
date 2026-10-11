# Status index

Point-in-time session reports. Open reports hold unmarked items (absence of
a marker is the open signal); annotated rows carry `done at` / `Won't
implement` / routed verdicts. Re-verify claims against current code before
acting on them.

## Open reports

| Date       | Report                                                                                                  | State                                                                                 |
| ---------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| 2026-09-24 | [paperlesstest session brutal review](2026-09-24_00-08_paperlesstest-session-brutal-review.md)          | Terminal report of the paperlesstest arc; open items are consumer-repo + billing work |
| 2026-10-11 | [branded-ID migration](2026-10-11_03-23_branded-id-migration-status.md)                                 | Annotated 2026-10-11 (docs-health); v0.5.0 release-ready, tag pending owner call      |
| 2026-10-11 | [branching-flow findings triage](2026-10-11_03-41_branching-flow-findings-triage.md)                    | Annotated EXECUTED + HARVESTED (executed by the 05:02 session)                        |
| 2026-10-11 | [branching-flow execution](2026-10-11_05-02_branching-flow-execution-status.md)                         | Annotated + harvested 2026-10-11 (docs-health)                                        |
| 2026-10-11 | [full-code-review session](2026-10-11_06-36_full-code-review-session-status.md)                         | Annotated + harvested 2026-10-11 (docs-health)                                        |
| 2026-10-11 | [docs-health audit: full backlog dispatch](2026-10-11_09-37_docs-health-audit-full-backlog-dispatch.md) | This pass's report: 19 archives, living-docs polish, dprint incident, owner questions |

## Archived (fully resolved — every item carries a verdict)

See [archived/](archived/). Bulk-archive manifest (2026-10-11 docs-health
pass; classification per the docs-health ANNOTATE table):

| File (in `archived/`)                                            | Classification         | Deciding reason                                                                            |
| ---------------------------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------ |
| 2026-09-13_13-42_docs-health-audit.md                            | ARCHIVE (pre-existing) | Every thread terminated (first docs-health pass); rows normalized to uniform strikes       |
| 2026-09-13_13-51_pareto-plan-sdk-hardening-and-release.md        | ARCHIVE (plan)         | T01–T24 all resolved (T06/T23 closed via v0.4.0); Resolution appendix added                |
| 2026-09-13_15-39_pareto-execution-v011-release-p3-batch.md       | ARCHIVE                | All items carried verdicts after the last three open rows were routed/Won't (this pass)    |
| 2026-09-13_18-37_18-skills-full-audit-and-hardening.md           | ARCHIVE                | Remaining f-rows resolved (README constants done; hooks/docs-site routed)                  |
| 2026-09-13_18-40_pareto-v030-landing-and-hardening.md            | ARCHIVE (plan)         | 26/26 coarse tasks resolved (4 gated rows closed later); Resolution appendix added         |
| 2026-09-14_03-29_full-execution-v020-release-t24-slice.md        | ARCHIVE                | 9 open f-rows routed/Won't; everything else already struck                                 |
| 2026-09-14_04-13_post-v0.3.0_build-gate-repair.md                | ARCHIVE                | 21 open rows resolved (multi-system decision landed in 0.4.0; rest routed/Won't)           |
| 2026-09-14_05-24_v030-execution-consumers-and-hardening-tail.md  | ARCHIVE                | 25 open rows resolved (v0.4.x shipped the routed TODO rows; rest routed consumer/Won't)    |
| 2026-09-14_07-23_docs-health-audit-annotate-archive-pass.md      | ARCHIVE                | Previous docs-health pass; its open items closed by this pass (erraudit, INDEX, leftovers) |
| 2026-09-14_09-32_v031-release-cut-and-publish.md                 | ARCHIVE                | Release residue resolved (indexing self-healed; ritual supersedes the manual checks)       |
| 2026-09-14_10-00_error-handling-hardening-session-report.md      | ARCHIVE                | 72 items annotated this pass (CI decision recorded; catalog enforced by drift test)        |
| 2026-09-16_17-34_deps-latest-verification-self-review-status.md  | ARCHIVE                | 70 items annotated (env/fleet items routed or moot via the govalid skip)                   |
| 2026-09-16_17-35_dependency-upgrade-status.md                    | ARCHIVE                | Continuation-annotated file; remaining rows closed (v0.3.2 shipped; 0.4.0 landed the rest) |
| 2026-09-16_17-38_buildflow-pipeline-repair-toolchain-and-gate.md | ARCHIVE                | 73 items annotated (upstream/machine routed; repo items shipped or moot)                   |
| 2026-09-16_17-57_todo-list-full-execution.md                     | ARCHIVE                | 58 items annotated (coverage/fuzz/docs items shipped in 0.4.x; ideas → ROADMAP)            |
| 2026-09-16_18-18_dependency-upgrade-full-session-status.md       | ARCHIVE                | Final-tree verification report; all routed items landed                                    |
| 2026-09-16_20-59_todo-rows-flake-hardening-coverage-status.md    | ARCHIVE                | 62 items annotated (all-systems/simd/fuzz shipped in 0.4.0; ideas → ROADMAP)               |
| 2026-09-23_16-19_paperlesstest-consumer-testing-sdk.md           | ARCHIVE (plan)         | EXECUTED end-to-end 2026-09-23; DoD fully ticked; Resolution appendix added                |
| 2026-09-23_18-12_paperlesstest-execution-status.md               | ARCHIVE                | Superseded mid-execution snapshot; every open bullet resolved by the superseding reports   |
| 2026-09-23_22-45_paperlesstest-consumers-done-status.md          | ARCHIVE                | Superseded; its single open item (CI billing) is routed to TODO_LIST                       |

Planning docs that remain: [2026-10-11 full-code-review plan](../planning/2026-10-11_06-00_full-code-review-plan.html)
(point-in-time input to the 06:36 review session).
