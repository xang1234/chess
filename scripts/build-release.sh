#!/bin/bash
set -euo pipefail

repository_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repository_root"

signing_identity=${CHESS_TRAINER_SIGNING_IDENTITY:?'CHESS_TRAINER_SIGNING_IDENTITY must name a Developer ID Application certificate'}
notary_profile=${CHESS_TRAINER_NOTARY_PROFILE:?'CHESS_TRAINER_NOTARY_PROFILE must name a notarytool keychain profile'}
credential_home=${HOME:?'HOME must locate the Apple signing and notarization credentials'}

release_tmp=$(mktemp -d "${TMPDIR:-/tmp}/chess-trainer-release.XXXXXX")
cleanup_release_tmp() {
  chmod -R u+w "$release_tmp" 2>/dev/null || true
  rm -rf "$release_tmp"
}
trap cleanup_release_tmp EXIT

export CHESS_TRAINER_RELEASE_ROOT="$release_tmp"
export GOMODCACHE="$release_tmp/go-mod-cache"
export GOCACHE="$release_tmp/go-build-cache"
export npm_config_cache="$release_tmp/npm-cache"
export GOWORK=off
export GOTOOLCHAIN=local
export GOENV=off
export GOFLAGS=
export GOCACHEPROG=
export NODE_OPTIONS=
export NODE_PATH=
export npm_config_node_options=
export npm_config_userconfig=/dev/null
export npm_config_globalconfig=/dev/null
export npm_config_script_shell=/bin/sh
export HOME="$release_tmp/home"
export TMPDIR="$release_tmp/tmp"
unset CC CXX CGO_CFLAGS CGO_CPPFLAGS CGO_CXXFLAGS CGO_LDFLAGS
unset GOOS GOARCH GOAMD64 GOARM64 GOFIPS140 GOEXPERIMENT
unset SDKROOT DEVELOPER_DIR MACOSX_DEPLOYMENT_TARGET
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
unset GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES
unset GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM
unset GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS
mkdir -p \
  "$GOMODCACHE" \
  "$GOCACHE" \
  "$npm_config_cache" \
  "$HOME" \
  "$TMPDIR"

tag=${1:?usage: scripts/build-release.sh <public-tag>}
release_version=${tag#v}
commit=$(git rev-parse HEAD)

test "$(go env GOVERSION)" = go1.26.4

release_source="$release_tmp/app-source"
release_source_tar="$release_tmp/app-source.tar"
mkdir -p "$release_source"
git archive --format=tar --output "$release_source_tar" "$commit"
tar -xf "$release_source_tar" -C "$release_source"

(
  cd "$release_source"
  npm --prefix frontend ci
  npm --prefix frontend run build
  go mod download all
  go mod verify
  npm --prefix frontend run verify:licenses
  node "$repository_root/scripts/verify-release.mjs" \
    --phase pre \
    --tag "$tag" \
    --input-root "$release_source"

  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build \
    -clean \
    -trimpath \
    -platform darwin/arm64 \
    -m \
    -nosyncgomod \
    -ldflags "-X chess-trainer/internal/buildinfo.Commit=${commit}"
)

release_app="$release_source/build/bin/Chess Trainer.app"
notary_archive="$release_tmp/Chess-Trainer-${tag}-macOS-arm64-notary.zip"
plutil \
  -replace CFBundleShortVersionString \
  -string "$release_version" \
  "$release_app/Contents/Info.plist"
plutil \
  -replace CFBundleVersion \
  -string "$release_version" \
  "$release_app/Contents/Info.plist"
HOME="$credential_home" codesign \
  --force \
  --options runtime \
  --timestamp \
  --sign "$signing_identity" \
  "$release_app"
ditto -c -k --keepParent "$release_app" "$notary_archive"
HOME="$credential_home" xcrun notarytool submit \
  "$notary_archive" \
  --keychain-profile "$notary_profile" \
  --wait
xcrun stapler staple "$release_app"
xcrun stapler validate "$release_app"
spctl --assess --type execute --verbose=4 "$release_app"

source_archive="$repository_root/build/release/Chess-Trainer-${tag}-corresponding-source.tar.gz"
node "$repository_root/scripts/build-corresponding-source.mjs" \
  --tag "$tag" \
  --commit "$commit" \
  --output "$source_archive"

final_app="$repository_root/build/bin/Chess Trainer.app"
mkdir -p "$(dirname "$final_app")"
rm -rf "$final_app"
ditto "$release_source/build/bin/Chess Trainer.app" "$final_app"

node "$repository_root/scripts/verify-release.mjs" \
  --phase post \
  --tag "$tag" \
  --input-root "$release_source" \
  --app "$final_app" \
  --source "$source_archive"

binary_archive="$repository_root/build/release/Chess-Trainer-${tag}-macOS-arm64.zip"
rm -f "$binary_archive"
ditto -c -k --keepParent "$final_app" "$binary_archive"
