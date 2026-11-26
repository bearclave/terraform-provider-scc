#!/usr/bin/env bash
set -euo pipefail
# Release of bearclave/scc by the owner from a clean main: tools/release.sh vX.Y.Z (needs go, goreleaser, gpg, az, gh).
version=${1:?usage: tools/release.sh vX.Y.Z}
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not vX.Y.Z: $version" >&2; exit 1; }
[ "$(git branch --show-current)" = main ] || { echo "run from main" >&2; exit 1; }
git fetch -q origin && [ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || { echo "main is not origin/main" >&2; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "working tree not clean" >&2; exit 1; }

for tool in go goreleaser gpg az gh; do
  command -v "$tool" >/dev/null || { echo "missing: $tool (brew install $tool)" >&2; exit 1; }
done

# The tag is signed with the owner's own git key (versioning.md#tags),
# before the throwaway keyring below replaces his. A rerun after a failed
# build finds the tag on HEAD and goes on.
if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
  [ "$(git rev-parse "$version^{commit}")" = "$(git rev-parse HEAD)" ] || { echo "$version exists on another commit" >&2; exit 1; }
else
  git tag -s "$version" -m "$version"
  git push origin "$version"
fi

# A throwaway keyring: the signing key never stays on the Mac.
GNUPGHOME=$(mktemp -d /tmp/gpg.XXXXXX); export GNUPGHOME
cleanup() { gpgconf --kill gpg-agent 2>/dev/null || true; rm -rf "$GNUPGHOME"; }
trap cleanup EXIT
az keyvault secret show --vault-name kv-tech-sec-weu-001 --name tf-provider-gpg-key --query value -o tsv | gpg --batch --quiet --import
GPG_FINGERPRINT=$(az keyvault secret show --vault-name kv-tech-sec-weu-001 --name tf-provider-gpg-fingerprint --query value -o tsv)
export GPG_FINGERPRINT

GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
