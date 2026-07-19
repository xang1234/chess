# Chess Trainer

Chess Trainer is a calm, local-first macOS chess-learning app designed for a child to
use without a browser account, advertisements, or a hosted service. It imports
local Lichess zstd/CSV, tactical PGN, canonical JSON, Lucas `.fns`, and linear
FEN/UCI puzzle collections, keeps learner history on the Mac, and includes a
**Learn Openings** workspace for guided lessons, spaced review, depth controls,
and read-only variation exploration. Opening material is supplied as an
external private `.ctcourse` pack; no commercial opening book or private course
is bundled with the application. A basic PGN game library is also included.
The board uses Chessground for familiar click, drag, selection, and move-marker
behavior.

The production application is a Wails desktop bundle. It does not start an HTTP
listener or require an internet connection at runtime. User data lives under
`~/Library/Application Support/Chess Trainer/`. Network access is only part of
installing dependencies and the deliberately gated public-release procedure.

## Develop and test

Install the locked frontend dependencies, verify the Go module cache, and run
the main checks from the repository root:

```bash
npm --prefix frontend ci
go mod download all
go mod verify
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
go test ./... -count=1
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
```

For live development, run the pinned Wails CLI:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev
```

See [the local build and acceptance guide](docs/operations/local-build.md) for
the complete test matrix, puzzle-import acceptance, recovery behavior, and the
ordinary local `.app` build. Local builds identify their source as
`development`. The [puzzle import format guide](docs/operations/puzzle-import-formats.md)
defines the supported grammars, source IDs, rating behavior, atomic replacement,
and exact resource limits.
The [opening-course authoring guide](docs/operations/opening-course-authoring.md)
defines the private course schema, validator, source-coverage checks, update
behavior, and the rule that private packs remain outside Git and releases.

## Public releases

The only supported distributable build entry point is:

```bash
scripts/build-release.sh <public-tag>
```

It intentionally fails unless the tree is clean, Go 1.26.4 is active, the
matching semantic-version tag is publicly reachable without credentials at
[github.com/xang1234/chess](https://github.com/xang1234/chess), and all release
inputs come from an isolated archive of the tagged commit and new disposable
caches. It also clears build-affecting Go, Node, npm, and Git environment
settings before creating both the signed-checkable macOS app and a
tag/commit-matched corresponding-source archive. Do not use it for an
unpublished local build. Follow [the release procedure](docs/operations/release.md)
before distributing a binary.

The module's `go.mod` declares the Go 1.25 language level. That is separate
from the mandatory Go 1.26.4 toolchain used for a distributable build and from
the reviewed Go `LICENSE` and `PATENTS` copies shipped with corresponding
source.

## License and source rights

Chess Trainer is free software licensed under the
[GNU General Public License, version 3 or later](LICENSE). You may use, study,
modify, and redistribute it under those terms. The preferred source for this
version is the public repository and, for each binary release, the matching
corresponding-source archive attached beside that binary.

THIS PROGRAM COMES WITH ABSOLUTELY NO WARRANTY, TO THE EXTENT PERMITTED BY LAW.

Bundled dependency terms and copyright notices are in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). In particular:

- Chessground 10.1.1 is GPL-3.0-or-later software by the Lichess Team. Its
  committed preferred source is
  [third_party/source/chessground-v10.1.1.tar.gz](third_party/source/chessground-v10.1.1.tar.gz).
- The bundled Nunito font is covered by the
  [SIL Open Font License 1.1](frontend/src/assets/fonts/OFL.txt).
- The committed Svelte preferred source is
  [third_party/source/svelte-v3.59.2.tar.gz](third_party/source/svelte-v3.59.2.tar.gz).

No chess engine is included. Any future engine feature must connect to an
external engine through a separately defined adapter; engine implementation is
outside this project.
