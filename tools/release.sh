#!/usr/bin/env bash
# Release of bearclave/scc: signed tag, goreleaser builds and signs the
# checksums with the key from kv-tech-sec-weu-001, the Terraform Registry
# picks the GitHub release up. Run by the owner from a clean main:
#   az-bearclave-matej && tools/release.sh v1.0.0
# Needs goreleaser, gpg, az (an account that reads kv-tech) and gh.
set -euo pipefail
version=${1:?usage: tools/release.sh vX.Y.Z}
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not vX.Y.Z: $version" >&2; exit 1; }
[ "$(git branch --show-current)" = main ] || { echo "run from main" >&2; exit 1; }
git fetch -q origin && [ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || { echo "main is not origin/main" >&2; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "working tree not clean" >&2; exit 1; }

# The tag is signed with the owner's own git key (versioning.md#tags),
# before the throwaway keyring below replaces his.
git tag -s "$version" -m "$version"
git push origin "$version"

# A throwaway keyring: the signing key never stays on the Mac.
GNUPGHOME=$(mktemp -d /tmp/gpg.XXXXXX); export GNUPGHOME
cleanup() { gpgconf --kill gpg-agent 2>/dev/null || true; rm -rf "$GNUPGHOME"; }
trap cleanup EXIT
az keyvault secret show --vault-name kv-tech-sec-weu-001 --name tf-provider-gpg-key --query value -o tsv | gpg --batch --quiet --import
GPG_FINGERPRINT=$(az keyvault secret show --vault-name kv-tech-sec-weu-001 --name tf-provider-gpg-fingerprint --query value -o tsv)
export GPG_FINGERPRINT

GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
