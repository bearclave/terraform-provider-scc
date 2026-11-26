# Versioning and releases

<!-- Standard of bearclave/governance (standards/versioning.md). In other repositories this is a managed copy - change it in governance. -->

A version exists where something is **consumed by reference** or **handed
over**. Configuration that is applied from `main` has no version: `main` is
what runs, and the apply run is its record.

## Which repositories release

| Repository | Versioned | Consumers pin |
|---|---|---|
| `ci` | yes | `@v2` (major tag), production repositories the exact `@v2.3.1` |
| `iac-modules` | yes | `modules_ref` in `account.hcl` of every Terragrunt repository, always a tag `vX.Y.Z` |
| `terraform-provider-scc` | yes | the `bearclave/scc` version in `account.hcl` of `iac-purview`, from the Terraform Registry |
| `template-iac-*` | yes | the version handed over to a customer, written into the handover |
| `app-home` | yes | the image `:vX.Y.Z` its release publishes; a release is a promotion ([below](#applications)) |
| `app-web-*` | yes | nobody pins: `prod` is what runs, the tag records each promotion ([below](#applications)) |
| `app-office-templates` | yes | the generated templates handed to the website |
| `vs-code-extensions` | per extension | the `version` in each extension's `package.json`, built into its VSIX; no git tag ([below](#vs-code-extensions)) |
| `iac-*` but `iac-modules` | no | applied from `main` |
| `governance`, `handbook`, `ops-tools`, `.github`, `content-sentinel` | no | read, not consumed |

## Repositories without a tag yet

`app-office-templates` and the `template-iac-*` repositories are versioned by
this standard but have no tag yet. Each gets `v1.0.0` with the next change a
consumer receives: `app-office-templates` when it next hands templates to the
website, a template at its next handover. With no earlier tag the first version is
`v1.0.0`; after it, the rules of [Releasing](#releasing) apply.

## When to release

An agent applies this rule after every merge it was part of, and the owner
before a consumer needs a change:

| Repository | Release when | Who |
|---|---|---|
| `ci` | a merged pull request since the last tag has a release effect (below) and a caller needs it - **at most once a week**, batching what was merged; a security fix or a broken pipeline at once. Every release reaches the pinned repositories as pull requests with a whole-tree plan | the owner runs `tools/release.sh`; `major-tag.yml` and `ci-pins.yml` follow |
| `iac-modules` | a consumer is to receive a module change: release first, then bump `modules_ref` in the consumer by hand (Dependabot does not read it), its plan in that pull request | the owner |
| `app-home`, `app-web-*` | the change is to run in production: `promote` ([below](#applications)) | the owner starts `promote`; the App `bearclave-promote` tags |
| `app-office-templates` | the templates are handed to the website | the owner |
| `template-iac-*` | before a handover to a customer | the owner |
| `vs-code-extensions` | in the pull request that changes an extension: its `package.json` version is bumped there | the author |

- No release for `docs`, `test`, `ci` and `chore` alone.
- An agent never tags or publishes a release (the settings deny `git tag`,
  `gh release create`): it says in its reply that a release is due, with the
  version worked out below and the commands of [Releasing](#releasing).

## Version numbers

[Semantic Versioning 2.0](https://semver.org/): `vMAJOR.MINOR.PATCH`, always
with the `v`.

The next version follows from the pull request titles merged since the last
tag (see [workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#commits)):

- any `!` / `BREAKING CHANGE` → **major** (`v2.4.1` → `v3.0.0`)
- otherwise any `feat` → **minor** (`v2.4.1` → `v2.5.0`)
- otherwise `fix`, `perf`, `refactor`, `deps`, `revert` → **patch** (`v2.4.1` → `v2.4.2`)
- only `docs`, `test`, `ci`, `chore` → no release needed

Before `v1.0.0` (a new repository), breaking changes bump the minor.

## Tags

- **Release tag** `vX.Y.Z`: annotated, signed (`tag.gpgsign`), on a commit of
  `main`, never moved or deleted. `gh release create` for a tag that does not
  exist makes a lightweight, unsigned tag on GitHub: never used;
  `major-tag.yml` and `ci-pins.yml` refuse such a release. The tag ruleset in `iac-github` refuses an
  update or deletion. An application's tag is the exception: annotated, made
  by the App `bearclave-promote` and not signed by the owner, on the commit `prod` was
  moved to ([below](#applications)).
- **Major tag** `vX` (only in `ci`): moves to the newest `vX.Y.Z`, so that
  callers on `@v2` get fixes. Only the owner and the GitHub App
  `bearclave-bot` may move it (tag ruleset); `vX` is a lightweight ref to
  the release's commit.
- No other tags. No `latest`, no date tags.

## Releasing

A release is made by the owner, by hand, with a tag signed by his key; a
workflow cannot sign as him, and the ruleset `version-tags-creation` lets
only organisation owners create `v*` tags. Applications are released by
`promote` instead ([Applications](#applications)).

1. Merge everything that belongs to the release.
2. Work out the version and preview the notes: the workflow `release.yml` of
   `bearclave/ci` in its default dry run (where a repository has a caller), or
   by hand from the titles since the last tag:
   `gh pr list --repo bearclave/<repo> --state merged --search "merged:>$(git log -1 --format=%cs <last-tag>)"`.
   Publishing through `release.yml` is not used: it would create an
   unsigned tag, and no `RELEASE_TOKEN` secret exists.
3. Run `tools/release.sh <repo-path> [vX.Y.Z]` of `bearclave/ci` (`--dry-run`
   first); it is the only way to release, never a chain of `git tag`,
   `git push` and `gh release create`. It refuses when it is not run in a
   terminal, a `main` that is not a clean copy of `origin/main`, an open pull
   request with auto-merge on or approved and mergeable (it may be about to
   land), a release with nothing that calls for one and a version other than
   the computed one; it checks its tools before the tag, then signs, pushes
   and publishes after the version is typed. Tag only after the pull requests
   of the release are merged (`MERGED`, not auto-merge on): a tag cannot be
   moved.

   In `ci` the major tag then moves by itself: publishing the release runs
   `major-tag.yml`, which moves `v2` as `bearclave-bot` to a published,
   non-pre-release release with a verified signed tag, only forward. Check
   it with `gh run list --repo bearclave/ci --workflow major-tag.yml --limit 1`.
   The manual fallback, and the first tag of a new major (`v3`), is
   `tools/move-major-tag.sh v2.5.0` in `ci`, run by the owner.

4. The release notes are grouped by the `type:*` labels of the pull requests
   (`.github/release.yml`, managed): Breaking changes, Features, Fixes,
   Performance, Dependencies, Other. Edit the generated notes only to add
   upgrade steps for a breaking change.

## Applications

`app-web-bearclave`, `app-web-matejhrabalek` and `app-home` are released by
promoting a commit of `staging` to production
([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#applications-staging-and-prod)), never by hand.
The workflow `promote` (`bearclave/ci` `promote.yml`, run by the owner from
`staging`):

1. works out the version from the pull request titles since the last tag, by
   the rules above, or takes the one given; only `docs`, `test`, `ci` and
   `chore` since the last tag is refused unless a version is given;
2. checks that the staging run of the commit succeeded and deployed staging;
3. as the GitHub App `bearclave-promote`, fast-forwards `prod` to the commit;
4. waits for the production deployment of the commit and, only when it
   succeeded, creates the annotated tag `vX.Y.Z` on it and publishes the
   release with notes generated from the pull requests. A version is what
   runs in production; a failed production deployment gets no tag.

**An application's tag is an unsigned tag of the App**, not signed by the
owner: the release is a click, and a workflow cannot sign with the owner's
key. What vouches for it is the rulesets (only an owner and
`bearclave-promote` create `v*` tags in an application, only
`bearclave-promote` moves `prod`), the App being installed on the
applications only, and the staging and production runs it checked.
Every other versioned repository keeps the owner's signed tags. A rollback
is a revert and a new promotion, so a version is never re-tagged or reused.

## vs-code-extensions

Each extension is versioned on its own, SemVer in its `package.json`, by the
same rules as the titles above; the pull request that changes an extension
bumps its version, and its README table follows. The VSIX is built locally
(`tools/build.py`) and installed by that version; nothing is published to the
Marketplace and no git tag is made, because one tag cannot carry six
versions.

## Changelog

The GitHub releases are the changelog; no `CHANGELOG.md` is kept. A
repository whose consumers cannot read GitHub (templates handed to a
customer) writes the release notes into the handover instead.
