# Puzzle Import Quality Remediation Design

**Date:** 2026-07-18

## Purpose

Address the maintainability findings from the thermo-nuclear review without
changing supported puzzle formats, import semantics, catalogue behavior, or UI
capabilities. The remediation should delete scattered policy, make state and
wire contracts explicit, and restore the repository's 1,000-line file limit.

## Goals

- Give each puzzle adapter one canonical description of its format metadata.
- Keep source identity and source kind authoritative in the shared importer.
- Make invalid frontend import-session states unrepresentable.
- Remove the unused inspection field from terminal and polled job results.
- Split the import-job test module and enforce source-size limits repository-wide.
- Delete dead rating-query plumbing and an impossible linear-metadata limit path.

## Non-Goals

- Add or remove puzzle formats.
- Change detection grammar, source-ID resolution, ratings, progress semantics,
  cancellation, generational activation, or import reports.
- Dynamically download or register adapters at runtime.
- Replace the bounded canonical JSON framing implementation.
- Move frontend presentation policy wholesale into backend APIs.

## Adapter Descriptor

`PuzzleAdapter` will expose a descriptor instead of a bare format identifier.
The descriptor contains:

- stable `ImportFormat` value;
- user-facing format label used in import confirmation;
- canonical filename extension used only to resolve ambiguous content;
- native chooser filter description.

The descriptor is the canonical Go-side home for format metadata. Adapter
selection, support checks, extension tie-breaking, inspection completion, and
native file filters derive their values from registered descriptors. The
application service remains the one composition root that chooses which
adapters are installed.

`ImportInspection` gains the descriptor's display label. The frontend renders
that backend-provided label rather than maintaining a second label map. The
frontend retains a closed format type, but defines its format tuple once and
derives both the union and runtime decoder allowlist from that tuple.

Descriptor fields are configuration, not source identity. Import revalidation
continues comparing path, filename, format, source ID, source-ID origin, and
embedded source metadata. Replacement status and display labels remain
presentation data rather than authoritative identity fields.

## Authoritative Occurrence Provenance

Format decoders continue to produce canonical puzzle cores and format-specific
occurrence data. Immediately before catalogue insertion, `CollectionImporter`
sets every accepted occurrence's `SourceID` and `SourceKind` from the resolved
inspection and selected descriptor.

This removes repeated source stamping from the five adapters and ensures the
same values used to open the generation are used for every staged occurrence.
Tactical PGN and canonical JSON decoders may still retain expected inspection
data to validate embedded source declarations and source-level defaults; they
will not own the final occurrence provenance fields.

## Frontend Import Session States

`ImportSessionState` becomes a discriminated union. Each phase carries only the
data valid for that phase:

- `idle`: no selected collection or job;
- `selecting`: the prior selectable state, so chooser cancellation can restore it;
- `inspecting`: the chosen path while backend inspection runs;
- `ready`: a required `ImportInspection`;
- `starting`: the required inspection while job creation runs;
- `running`: inspection, non-empty job ID, and progress snapshot;
- `finished`: inspection, job ID, progress, and terminal result.

An error string remains available on each state. Type guards identify states
that permit file selection or import start. The Svelte panel renders inspection,
progress, and result data only after narrowing to a state that owns it.

File selection and backend inspection use separate error boundaries. Chooser
cancellation restores the prior selectable state. Inspection failure returns to
idle with the backend error, preserving current behavior without an auxiliary
`inspecting` boolean.

Events and polling continue to merge monotonic progress and ignore stale job
IDs. A terminal result may transition only the matching running job to finished.

## Import Result Contract

`importjob.Result` contains only job ID, status, progress, report, and optional
error. It no longer serializes `ImportInspection`. The running job receives the
confirmed inspection directly, and the frontend session already owns the
inspection associated with its job. Generated Wails bindings, test backends,
decoders, and fixtures will use the same reduced shape.

## Test Decomposition and Architecture Guard

`internal/importjob/service_test.go` is split by responsibility:

- import start, result, progress, and cancellation lifecycle tests;
- cleanup, writer exclusion, close, and concurrency-ordering tests;
- shared test doubles and bounded channel helpers.

No resulting file may exceed 1,000 lines.

A root-level architecture test walks owned `.go`, `.ts`, and `.svelte` source
files and fails when any exceeds 1,000 lines. It excludes generated bindings,
dependencies, build output, vendored source, Git metadata, and linked-worktree
administration. The package-local reflection test that bans boolean parser
fields is removed because it asserts an implementation shape rather than a
behavioral or architectural boundary.

## Direct Simplifications

`LearnerRatingBounds` stops selecting and scanning source kind because rating
bounds no longer depend on kind. The unnecessary `sources` join is removed,
while the existing per-generation indexed minimum/maximum strategy is retained.

Linear FEN/UCI metadata can contain only one machine-sized integer. Its JSON
encoding cannot approach 64 KiB, so the metadata-size constant, per-record
marshal, JSON import, and unreachable errors are removed.

## Error Handling and Compatibility

All user-visible import errors and accepted payload shapes remain unchanged
except that `Result.inspection` disappears from the Wails wire result. The app's
manual frontend result contract already omitted that field, so no rendered UI
behavior changes.

Existing unsupported-format, duplicate-adapter, and ambiguous-content errors
stay authoritative. Descriptor consumers reject missing descriptor fields at
the same boundary where those values are required.

## Verification

Implementation follows red-green-refactor cycles for:

1. descriptor-derived extension matching, chooser filters, and inspection labels;
2. runner-owned source ID and kind stamping;
3. reduced result JSON shape;
4. discriminated frontend state transitions and stale-event handling;
5. repository-wide source-size enforcement.

Existing catalogue and adapter integration tests verify behavior preservation
for the query and metadata simplifications. Final verification runs Go tests,
the race detector, vet, frontend unit and type checks, the production build,
license verification, Playwright, performance-tag compilation, and diff/status
checks.
