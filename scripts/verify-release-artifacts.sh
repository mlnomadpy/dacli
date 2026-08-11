#!/usr/bin/env bash
# Prove the release path WITHOUT publishing anything.
#
#   ./scripts/verify-release-artifacts.sh
#
# Builds a snapshot, then checks the things a consumer actually depends on:
# every supported platform is present, each archive has an SBOM, the checksums
# file verifies, and the binary a user would extract by following the README
# actually runs.
#
# Deliberately a SNAPSHOT. Publishing is a separate, human act — the maintainer
# pushes a `v*` tag and .github/workflows/release.yml takes it from there.
# Nothing here creates or pushes a tag, and this script is the evidence that
# when a tag IS pushed, the path works.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

command -v goreleaser >/dev/null || { echo "✗ goreleaser not installed"; exit 1; }
command -v syft       >/dev/null || { echo "✗ syft not installed (needed for SBOMs)"; exit 1; }

echo "▸ snapshot build"
goreleaser release --snapshot --clean --skip=homebrew >/dev/null 2>&1

fail() { echo "✗ $1"; exit 1; }

# 1. Every supported platform, or a consumer on the missing one has no install.
for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 windows_amd64 windows_arm64; do
  # Either extension counts: windows ships .zip, the rest .tar.gz. Globbing
  # both at once fails whenever one does not match, which is always.
  # || true: the assignment takes the LAST command's status, and one of the
  # two globs never matches, which under set -e would kill the script.
  found=$( { ls dist/*"${target}".tar.gz 2>/dev/null; ls dist/*"${target}".zip 2>/dev/null; } || true )
  [ -n "$found" ] || fail "no archive for ${target}"
done
echo "✓ 6 platform archives"

# 2. An SBOM per archive. A release note claiming contents is not evidence.
for a in dist/*.tar.gz dist/*.zip; do
  [ -f "${a}.sbom.json" ] || fail "no SBOM for $(basename "$a")"
done
echo "✓ an SBOM per archive"

# 3. The checksums file actually verifies. A checksums.txt nobody checks is
#    decoration.
( cd dist && shasum -a 256 -c checksums.txt >/dev/null 2>&1 \
    || sha256sum -c checksums.txt >/dev/null 2>&1 ) \
  || fail "checksums.txt does not verify"
echo "✓ checksums verify"

# 4. THE README PATH. Extract the way the README tells a user to, and run it.
#    This is the acceptance criterion: a tagged release must be installable and
#    exercisable using README commands only.
OS=$(go env GOOS); ARCH=$(go env GOARCH)
ARCHIVE=$(ls dist/*_"${OS}"_"${ARCH}".tar.gz 2>/dev/null | head -1)
[ -n "$ARCHIVE" ] || fail "no archive for this host (${OS}_${ARCH})"

TMP=$(mktemp -d)
tar xz -C "$TMP" -f "$ARCHIVE"          # the README's `| tar xz`
[ -x "$TMP/dacli" ] || fail "the archive contains no executable dacli"
"$TMP/dacli" version >/dev/null          || fail "the extracted binary does not run"
"$TMP/dacli" --help  >/dev/null          || fail "the extracted binary cannot print help"
echo "✓ the README install path yields a working binary ($(basename "$ARCHIVE"))"

echo
echo "✓ release artifacts verified — nothing published, no tag created"
