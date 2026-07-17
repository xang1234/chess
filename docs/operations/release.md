# Public macOS arm64 release procedure

This is the only supported procedure for distributing a Chess Trainer binary.
It keeps the executable, its exact public source identity, GPL notices, and
complete corresponding source tied to the same public tag. The binary produced
by this procedure targets Apple Silicon (`darwin/arm64`).

## Local builds are not public releases

The ordinary Wails command in [local-build.md](local-build.md) creates an
ad-hoc-signed development build for testing on the Mac that built it. An ad-hoc
signature has no Developer ID identity and does not satisfy this public-release
procedure. Do not rename, archive, or publish a local build as a release.

Only `scripts/build-release.sh <public-tag>` may produce a public binary. The
wrapper requires a Developer ID Application identity and an Apple notary
profile, builds the arm64 app, signs it with the hardened runtime and a trusted
timestamp, notarizes it, staples the ticket, and verifies Gatekeeper acceptance.

## Preconditions

- Use macOS with Xcode Command Line Tools and exactly Go 1.26.4.
- Install a valid Apple **Developer ID Application** certificate and its private
  key in a keychain available to `codesign`. Set the complete identity exactly
  as shown by the keychain, including the team identifier:

  ```bash
  export CHESS_TRAINER_SIGNING_IDENTITY='Developer ID Application: Example Name (TEAMID)'
  ```

  The wrapper rejects an absent identity, an identity of another certificate
  class, and ad-hoc signing (`-`).
- Create a `notarytool` keychain profile once. Supply an app-specific password
  at the prompt or through the supported `notarytool` credential options; do
  not commit credentials to this repository:

  ```bash
  xcrun notarytool store-credentials "ChessTrainer-Notary" \
    --apple-id "developer@example.com" \
    --team-id "TEAMID"
  export CHESS_TRAINER_NOTARY_PROFILE='ChessTrainer-Notary'
  ```

  `CHESS_TRAINER_NOTARY_PROFILE` is the profile name, not an Apple ID, password,
  or keychain secret. Confirm the profile belongs to the same Apple developer
  team as the signing identity.
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

4. Confirm the repository and tag are readable from a signed-out browser. In
   the same shell, export the Developer ID identity and notary profile described
   above, then run the supported wrapper:

   ```bash
   : "${CHESS_TRAINER_SIGNING_IDENTITY:?set the Developer ID Application identity}"
   : "${CHESS_TRAINER_NOTARY_PROFILE:?set the notarytool keychain profile}"
   scripts/build-release.sh "$tag"
   ```

   The wrapper exports the exact commit into an isolated tracked-source tree,
   creates new empty Go module, Go build, and npm caches under the same
   disposable release root, and installs/downloads only into those caches. It
   rejects untracked build inputs and inherited overlays/configuration,
   verifies the exact Go 1.26.4 toolchain and legal files, proves the fixed
   GitHub HTTPS tag resolves to `HEAD` without credentials, builds with the
   exact commit injected, and validates the generated corresponding-source
   archive. It then signs the arm64 app with the configured Developer ID
   Application identity, hardened runtime, and trusted timestamp. Build tools
   keep the disposable HOME, while signing and notarization alone use the
   invoking user's HOME so the login keychain and named notary profile remain
   available. Before signing, the wrapper derives `1.2.3` from a `v1.2.3` tag
   and writes it to both `CFBundleShortVersionString` and `CFBundleVersion`; the
   post-build verifier requires both values to match the tag. The wrapper then
   submits it with
   `xcrun notarytool submit ... --keychain-profile ... --wait`; staples and
   validates the accepted ticket; and requires both strict `codesign`
   verification and a successful Gatekeeper assessment before succeeding.

5. Confirm the wrapper-created binary archive exists and record its digest.
   Do not recreate or overwrite it outside the guarded wrapper:

   ```bash
   binary_archive="build/release/Chess-Trainer-${tag}-macOS-arm64.zip"
   test -f "$binary_archive"
   shasum -a 256 "$binary_archive"
   ```

6. Create the GitHub release for the same tag. Attach
   `Chess-Trainer-${tag}-macOS-arm64.zip` as the Apple Silicon binary.
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
- A missing, ad-hoc, non-Developer-ID, expired, or wrong-team signing identity
  must fail. Do not bypass the failure with a local signature.
- A missing or invalid notary profile, rejected or incomplete notarization,
  failed staple validation, or failed Gatekeeper assessment must fail. Inspect
  the `notarytool` result, correct the application or credentials, and submit a
  newly built release; do not publish an unstapled binary.
- A different Go version, shared cache, modified dependency extraction,
  truncated license, changed source archive, incomplete vendor/Wails source,
  missing legal asset, mismatched executable commit, non-arm64 executable, or
  missing hardened-runtime/timestamp properties or failed strict `codesign`
  verification must fail.

THIS PROGRAM COMES WITH ABSOLUTELY NO WARRANTY, TO THE EXTENT PERMITTED BY LAW.
