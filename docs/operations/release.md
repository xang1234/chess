# Public macOS release procedure

This is the only supported procedure for distributing a Chess Trainer binary.
It keeps the executable, its exact public source identity, GPL notices, and
complete corresponding source tied to the same public tag.

## Preconditions

- Use macOS with Xcode Command Line Tools and exactly Go 1.26.4.
- Ensure the intended commit is already public at
  `https://github.com/xang1234/chess`.
- Use a plain semantic version tag such as `v1.2.3`. Prerelease text, slashes,
  abbreviated commits, and local-only tags are deliberately rejected.
- Do not set a Go workspace or allow automatic toolchain download. The wrapper
  fixes `GOWORK=off`, `GOTOOLCHAIN=local`, and `GOENV=off` itself and clears
  inherited build-affecting Go, Node, npm, and Git settings.

## Build and publish

1. Run the complete local verification matrix in
   [local-build.md](local-build.md), including Go/race tests, frontend tests,
   browser tests, legal/source checks, and release-fixture tests.
2. Review and commit every intended application source, generated binding,
   legal asset, dependency lock, runtime notice lock, build script, and
   documentation change. `git status --short` must be empty.
3. Create the matching tag at that exact commit and push both commit and tag:

   ```bash
   tag=v1.2.3
   git tag "$tag"
   git push origin HEAD
   git push origin "$tag"
   ```

4. Confirm the repository and tag are readable from a signed-out browser, then
   run the supported wrapper:

   ```bash
   scripts/build-release.sh "$tag"
   ```

   The wrapper exports the exact commit into an isolated tracked-source tree,
   creates new empty Go module, Go build, and npm caches under the same
   disposable release root, and installs/downloads only into those caches. It
   rejects untracked build inputs and inherited overlays/configuration,
   verifies the exact Go 1.26.4 toolchain and legal files, proves the fixed
   GitHub HTTPS tag resolves to `HEAD` without credentials, builds with the
   exact commit injected, verifies bundled legal bytes and strict
   code-signature structure, and validates the generated corresponding-source
   archive before succeeding.

5. Archive the `.app` without changing its bundle contents:

   ```bash
   binary_archive="build/release/Chess-Trainer-${tag}-macOS.zip"
   ditto -c -k --sequesterRsrc --keepParent \
     "build/bin/Chess Trainer.app" "$binary_archive"
   ```

6. Create the GitHub release for the same tag. Attach
   `Chess-Trainer-${tag}-macOS.zip` as the binary.
7. Attach
   `Chess-Trainer-${tag}-corresponding-source.tar.gz` beside the binary. From a
   signed-out browser, download it successfully and inspect its
   `SOURCE_MANIFEST.json`, `TRACKED_FILES.sha256`, and `BUILDING.md`.
8. Retain GitHub's automatically generated tag-matched source ZIP and tarball
   on the same release page. These do not replace the corresponding-source
   archive, which also carries the production Go vendor/legal tree, complete
   Wails module source, exact Go legal files, and reviewed frontend sources.
9. Optionally attach the individual committed Chessground and Svelte preferred
   source archives. They are already included in the required corresponding-
   source artifact under `app/third_party/source/`.

## Expected failures

Do not bypass a verifier failure. Fix the named input, commit the correction,
create and push a new matching tag, and rerun the wrapper. In particular:

- An unpublished commit/tag or private-only authenticated `origin` must fail
  the credential-free fixed-GitHub check.
- Any dirty or untracked file must fail. If Wails regeneration changes tracked
  bindings, review and commit those changes, then retag; the wrapper never
  commits or hides generated drift.
- A different Go version, shared cache, modified dependency extraction,
  truncated license, changed source archive, incomplete vendor/Wails source,
  missing legal asset, mismatched executable commit, or failed strict
  `codesign` verification must fail.

THIS PROGRAM COMES WITH ABSOLUTELY NO WARRANTY, TO THE EXTENT PERMITTED BY LAW.
