---
paths:
  - ".github/**"
---

# Pipelines

<!-- Standard of bearclave/governance (standards/pipelines.md). In other repositories this is a managed copy - change it in governance. -->

All pipeline logic lives once, in [`bearclave/ci`](https://github.com/bearclave/ci),
as reusable workflows. A repository holds only short callers.

## The callers in a repository

| File | Trigger | Jobs and check names |
|---|---|---|
| `.github/workflows/ci.yml` | pull request, push to `main`, manual; in Terragrunt repositories also a submitted review; in an application pull request (also closed), push to `staging` and `prod`, in a container application also the push of a `v*` tag | `security` → **`security / scan`**; `pipeline` → its jobs and **`pipeline / gate`**; `review` (on a review only) re-runs `pipeline / destroy / approved` |
| `.github/workflows/pr.yml` | pull request opened, edited, synchronised, reopened, ready | `pr` → **`pr / check`** (managed file) |
| `.github/workflows/drift.yml` | weekly schedule, manual | `drift` → the drift run of the pipeline (only where something is applied) |
| `.github/workflows/promote.yml` | manual, on `staging` (applications only) | `promote` → the release to production ([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#applications-staging-and-prod)) |

The checks in bold are required on the default branch (`main`; `staging` in an application) by the organisation rulesets in
`bearclave/iac-github`. Every repository has all three; a repository with
nothing to build calls the `checks` pipeline, whose `gate` passes when its
checks do.

`ci.yml` passes the repository's variables and secrets explicitly
(`with:` / `secrets:`), never `secrets: inherit`. The caller owns the
triggers and the `concurrency` group; the shared workflow owns everything
else. Templates for each pipeline: [templates/workflows/](https://github.com/bearclave/governance/tree/main/standards/templates/workflows/).

## The shared workflows

| Workflow | Used by | Does |
|---|---|---|
| `security.yml` | every repository | gitleaks over the history, actionlint, zizmor, shellcheck |
| `pr.yml` | every repository | title and branch format, labels, assignee, description sections, root layout, `todo-issue`, Markdown links, the objects of collections against their schema, markdownlint |
| `checks.yml` | `checks` | ESLint where there is JavaScript, optional extra command, then `gate` |
| `terragrunt.yml` / `terragrunt-drift.yml` | `iac-azure`, `iac-github`, `iac-identity`, `iac-conditional-access`, `iac-intune`, `iac-exchange`, `iac-purview`, `iac-sentinel` | validate (fmt, HCL, `terraform validate`, lock files), conventions, Sentinel lint, plan of affected units with trivy, PR comment, `destroy / approved`, apply from `main`, verification of the apply, gate; drift fails on check blocks too |
| `destroy-review.yml` | the repositories of `terragrunt.yml` | on a submitted approval, re-runs the failed `destroy / approved` of the last pull request run |
| `terragrunt-home.yml` | `iac-home` | plan and apply of the router on the NAS runner (`network-runner`; through WireGuard from a GitHub runner as the fallback), certificate renewal, drift |
| `terraform-modules.yml` | `iac-modules` | fmt, validate, tflint, trivy config, conventions |
| `static-web-app.yml` | `app-web-*` | common site checks (with SEO), ESLint, the site's own checks, one build per commit (artifact `site`); a preview in the staging SWA per pull request, deploy to the staging SWA from `staging` (environment `staging`, noindex), the same build to production from `prod` (environment `prod`); smoke test after each |
| `promote.yml` | the applications | a commit of `staging` to production as the App `bearclave-promote`: checks its staging run, fast-forwards `prod`, waits for the production run, then tags `vX.Y.Z` and publishes the release |
| `container.yml` | `app-home` | ESLint, lint and tests in the image; with `environments`: push to `staging` publishes `:staging` and `:sha-<commit>` and deploys staging on the NAS runner, push to `prod` (or a manual run there) deploys that image without a rebuild, the release tag `vX.Y.Z` pushed by `promote.yml` adds `:vX.Y.Z` to it, a smoke test after each deployment; without: push to GHCR from `main` and from a release |
| `content-sentinel.yml` | `content-sentinel` until its catalogue is converted to the YAML of `iac-sentinel`, then `checks` | Sentinel lint, generated views up to date |
| conventions, Sentinel lint (fragments) | inside the pipelines above | rule checks driven by the repository's YAML |
| `release.yml` | versioned repositories, on demand | dry run: the next version and the release notes ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#releasing)) |
| `housekeeping.yml` | the organisation, weekly | stale branches and pull requests, one issue per repository ([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#housekeeping)) |

The workflows of `ci` itself, not called by other repositories:

| Workflow | Trigger | Does | Identity (environment) |
|---|---|---|---|
| `ci.yml`, `pr-check.yml` | pull request, weekly | tests, self-tests, generated workflows up to date; the `pr` check | `GITHUB_TOKEN` |
| `deps.yml` | weekly (actions), monthly (tools, providers) | the `deps` pull requests ([toolchain.md](https://github.com/bearclave/governance/blob/main/standards/toolchain.md#dependency-updates)) | `sp-ci-deps` → `bearclave-bot` (`deps`) |
| `ci-pins.yml` | a published release | the pin bumps in the repositories on `@vX.Y.Z` | `sp-ci-deps` → `bearclave-bot` (`deps`) |
| `major-tag.yml` | a published release | moves `vX` ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#tags)) | `sp-ci-major-tag` → `bearclave-bot` (`major-tag`) |
| `project-sync.yml` | daily 06:15 UTC | issues and pull requests into the Projects ([projects.md](https://github.com/bearclave/governance/blob/main/standards/projects.md)) | `sp-project-sync` → `bearclave-bot` (`project-sync`) |
| `housekeeping` | weekly | the housekeeping report for every repository | `bearclave-bot` |

Versions: callers use `@v2`; repositories with `touches-production=true`
pin the exact release (`@v2.3.1`) and `ci-pins.yml` in `ci` opens the bump
after each release ([toolchain.md](https://github.com/bearclave/governance/blob/main/standards/toolchain.md#dependency-updates)), so a
change to an apply pipeline reaches production only through a pull request
with a plan. See [versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md).

A repository in another organisation (a customer's, [customers.md](https://github.com/bearclave/governance/blob/main/standards/customers.md))
cannot reach the private `bearclave/ci`: it carries local copies of the
shared workflows and calls them with `uses: ./.github/workflows/...`, with
the same check names.

## Checks and their exceptions

Every check of the organisation, the pipeline that runs it, where it is
configured and where an exception goes. Nothing is checked outside this
table; a new check comes with its row.

| Check | Pipeline (job) | Config | Exceptions |
|---|---|---|---|
| gitleaks | `security` (`scan`) | `.gitleaks.toml` (default rules + allowlists) | `[[allowlists]]` in `.gitleaks.toml`, `description` = reason + `review:` line |
| actionlint | `security` (`scan`) | - | none: fix the workflow |
| zizmor | `security` (`scan`) | in `security.yml` | none in a repository |
| shellcheck | `security` (`scan`), and inside `run:` through actionlint | `--severity=warning` | `# shellcheck disable=SCxxxx # reason` on the line |
| markdownlint-cli2 | `pr` (`check`) | `src/markdownlint.yaml` of ci; `.github/markdownlint.yaml` of the repository replaces rules one by one | a rule set in `.github/markdownlint.yaml` with a comment why, or `<!-- markdownlint-disable-next-line MDxxx -->` with a reason |
| `markdown-links` (relative links and anchors in `*.md`) | `pr` (`check`) | - | none: fix the link |
| `todo-issue` (`TODO`/`FIXME` with an issue number) | `pr` (`check`) | `.github/conventions.yaml` | `exceptions` in `.github/conventions.yaml` |
| `collection-schema` (every `<key>.yaml` of a collection against the JSON Schema of its module) | `pr` (`check`) | `.github/schemas/<module>.schema.json`; `collection-schema` in `.github/conventions.yaml` | `exceptions` in `.github/conventions.yaml` |
| title, branch, description, labels | `pr` (`check`) | `src/pr_check.py` of ci | none |
| root layout | `pr` (`check`) | `standards/layout.yaml` | `layout.exceptions` in `.github/conventions.yaml` (`review` optional) |
| conventions | `terragrunt`, `terragrunt-home`, `terraform-modules` (`conventions`) | `.github/conventions.yaml` `rules` | `exceptions` in `.github/conventions.yaml` |
| Sentinel lint | `terragrunt`, `content-sentinel` (`sentinel`) | `.github/sentinel-lint.yaml` | `exceptions` in `.github/sentinel-lint.yaml` |
| exception reviews | the conventions and Sentinel jobs, `security` for gitleaks | the files of this column | - |
| `terragrunt hcl fmt`, `hcl validate`, `terraform validate`, lock files | `terragrunt` (`validate`); `hcl fmt` in `terragrunt-home` | - | none |
| trivy over the plans | `terragrunt`, `terragrunt-drift` (`plan`) | HIGH, CRITICAL | `.github/trivyignore.yaml`, `paths` = `<unit>/tfplan.json` |
| `terraform fmt` / `validate` | `terraform-modules` (`terraform`) | - | none |
| tflint | `terraform-modules` (`terraform`) | `.tflint.hcl` (managed, below) | a rule switched off in the repository section of `.tflint.hcl`, or `# tflint-ignore: <rule> # reason, review YYYY-MM-DD` above the block |
| trivy config | `terraform-modules` (`terraform`) | HIGH, CRITICAL | `.github/trivyignore.yaml` |
| ESLint | `static-web-app`, `container`, `checks` (`eslint`) | `eslint.config.mjs` (managed, below) | the repository section of `eslint.config.mjs`, each entry with a comment why |
| common site checks | `static-web-app` (`site`) | `src/staticwebapp.config.json` | none: fix the site |
| site checks of the repository | `static-web-app` (`site`) | `site-checks` input | in the repository's script |
| ruff, tests | `container` (in the `Dockerfile`) | `[tool.ruff]` in `pyproject.toml` | `[tool.ruff.lint]` of the repository |

**One exception format.** Wherever the format is ours, an exception names
what (`rule`; `id` in trivy), where (`path`; `paths` in trivy), why
(`reason`; `statement` in trivy) and when it is looked at again: `review`,
a date (YYYY-MM-DD), six months ahead when it is written. A missing reason
or a date that is not a date fails the check; a review date that has
passed, or a missing one, is a warning, and the exception stays in force until
someone decides: renew the date with a sentence on why it still holds, or
fix the finding and remove the exception.

```yaml
# .github/conventions.yaml (sentinel-lint.yaml: rule = the check, path = the rule file)
exceptions:
  - rule: no-hardcoded-location
    path: infrastructure/mg-corp/global/policy-assignments/terragrunt.hcl
    reason: The allowed-locations policy lists the regions it allows.
    review: 2027-05-19

# .github/trivyignore.yaml
misconfigurations:
  - id: AVD-AZU-0013
    paths: [infrastructure/mg-corp/.../key-vault/tfplan.json]
    statement: Why the finding is accepted.
    review: 2027-05-19
```

```toml
# .gitleaks.toml
[[allowlists]]
description = """
Why this is not a secret.
review: 2027-05-19
"""
regexes = ['''...''']
```

**tflint** runs where the Terraform is: in `terraform-modules`. A
Terragrunt repository holds units (`terragrunt.hcl`) and no `.tf` of its
own - its units take every module from `iac-modules` at `modules_ref`, and
that module was linted there, with the same base config, before the ref
existed. So tflint does not run in `terragrunt*`; what reaches the units is
covered by `hcl validate`, the plan and trivy over it.

**Managed lint configs**, written by `tools/sync.py <repository>` like
`.gitattributes`: the base above the marker is the same everywhere, the
section below it belongs to the repository and is kept.

| File | Written for | Base |
|---|---|---|
| `.tflint.hcl` | repositories with `*.tf` (not templates) | the `terraform` preset `recommended`, the `azurerm` ruleset at a pinned version, `azurerm_resources_missing_prevent_destroy` on |
| `eslint.config.mjs` | repositories with `*.js`/`*.mjs`/`*.cjs` (not templates) | `@eslint/js` recommended, browser and Node globals |

ESLint runs the version pinned in the repository's `package.json`
`devDependencies` where there is one, otherwise the version pinned in
`bearclave/ci` through `npx`. Lint errors that are real bugs are fixed;
everything else is configured in the repository section, not rewritten.

## Rules for every workflow

- **Plan on the pull request, apply from `main`.** Apply runs in a protected
  environment (`<subject>-prod` of the repository,
  [repo-types.md](https://github.com/bearclave/governance/blob/main/standards/repo-types.md#iac-terragrunt-terragrunt-home)) whose deployment branch policy allows `main`
  only. An application deploys to `staging` from `staging` and to `prod`
  from `prod`, each environment allowing its branch only. The plan and the drift sign in as the read-only identity, the apply
  as the write identity ([identities.md](https://github.com/bearclave/governance/blob/main/standards/identities.md)).
- **Apply only what the plan proposed**, and only when it proposed
  something.
- **Destroys and replaces need an approval given after the plan.** On a pull
  request whose plan destroys or replaces anything, `destroy / approved`
  (part of the gate) passes only with an approving review submitted after
  the plan job ended; a submitted review re-runs it (`review` job of
  `ci.yml`, `pull_request_review: submitted`). On `main` the push applies the
  destroys when the merged pull request passed that check on an approval;
  otherwise the run is red and waits for the manual run with `allow_destroy`,
  the fallback. A plan without destroys passes the check at once.
- **A due apply that did not happen is red.** Every pipeline ends with a
  `gate` job (`if: always()`) that checks the results of what it needs.
  A skipped job passes a required check, so the gate - never a skippable
  job - is the required one.
- **No `paths` filter on `pull_request`** of a workflow with a required
  check: a pull request it never runs for waits forever. Skip inside
  instead (a `changes` job).
- `concurrency`: a pull request cancels its previous run; a review run has
  a group of its own, so that approving never cancels a plan; runs on
  `main`, manual runs and drift share one group per state and never overlap.
- `timeout-minutes` on every job; `permissions` minimal, at job level.
- Tools as pinned release binaries checked against sha256; actions pinned
  to a commit SHA ([security.md](https://github.com/bearclave/governance/blob/main/standards/security.md)).
- The plan goes into one pull request comment that is rewritten on every
  push, and into the job summary; raw plan files stay on the runner (see
  Pull request comments).
- Drift runs weekly, plans the whole scope and fails on any difference.
- **Validate before the plan.** Formatting, syntax and `terraform validate`
  run in a job of their own, without any sign-in, and the plan needs it: a
  mistake fails within a minute with an annotation saying what to do. A
  committed `.terraform.lock.hcl` must not change in `init`.
- **Verify after the apply, never roll back by itself.** Right after the
  apply: a second plan of the applied units must propose nothing, the apply
  identity must have no failed operation in the Azure Activity Log of the
  affected subscriptions (where there are any), and no `check {}` block may
  fail. A failure makes the run red and the pipeline opens a `Bug` issue
  (`prio:high`, `source:drift`) and a draft revert pull request
  (`revert/<short-sha>`, `revert: <original title>`); the owner decides.
  The results go to the job summary and `verification.json`, for the
  `### Verification` section of the apply comment. Only the job that does
  this has `contents`, `pull-requests` and `issues: write`, so the caller
  grants them to `pipeline`.

## Pull request comments

A pipeline that plans speaks to the pull request in two comments, the same
in every repository. Both are deterministic, built from the plan in JSON by
the scripts of `bearclave/ci` (README, Plan and apply comments), never by a
model.

**Plan comment** - one per pull request, rewritten on every push:

1. Heading `## 📋 Plan complete` or `## ❌ Plan failed`.
2. One line: repository · commit · units · ➕ created · ✏️ updated ·
   🗑️ destroyed · ♻️ replaced (and 📥 imported, 🚚 moved, ⏏️ forgotten
   when there are any).
3. `### ⚠️ Destroys and replaces` first, when there are any:
   icon | Resource | Name | Unit | Why (for a replace, the setting that
   forces it).
4. `### Changes`: icon | Resource | Name | What changes | From | To, one row
   per change that matters, in fixed wording - ``tag `owner` ``,
   ``setting `x` ``, ``setting `x`: added `y` ``, ``diagnostic settings
   added: logs `...` → workspace `...` ``. Resource types read as names
   (Key Vault, Repository).
5. Imports that change nothing as one count line.
6. **Status** (security findings, formatting, validation) and **Needed**
   (what has to happen: an approval of a destroy, a fix, review and merge).
7. Terraform's own diff per unit, folded in `<details>`.

Only what really changes is shown: values known only after apply, computed
IDs, etags, URLs, timestamps, data sources and outputs are left out. Over
GitHub's size limit the details are cut, never the tables, and the comment
says what was cut. The raw plan is in the run artifact.

**Apply comment** - on the pull request merged as the applied commit, after
each apply on `main`: `## ✅ Apply complete` or `## ❌ Apply failed`;
repository · merge commit · units · duration; icon | Resource | Name |
What happened | Result, the plan's rows in the past tense with ✅ done,
❌ failed or ⏭️ not applied; errors; `### Verification`, filled by the
post-apply checks.

## Changing a shared workflow

1. Change it in `bearclave/ci` (template or script, then `tools/build.py`),
   test it from the pull request's own run.
2. Merge, release (`v2.x.y`, move `v2`).
3. Repositories on `@v2` get it at once; pinned repositories get a
   `bearclave-bot` pull request (`ci-pins.yml`) with a plan.
