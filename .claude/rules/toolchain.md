---
paths:
  - ".github/**"
  - "**/*.hcl"
  - "**/*.tf"
---

# Toolchain

<!-- Standard of bearclave/governance (standards/toolchain.md). In other repositories this is a managed copy - change it in governance. -->

What the pipelines run on, which tools and providers they use, where each
version is pinned and how it is bumped. The goal is a small, boring
toolchain: one runner image, as few tools as possible, every version pinned
where its pipeline reads it.

## Runners

| Rule | Why |
|---|---|
| GitHub-hosted runners, `runs-on: ubuntu-24.04` | a fixed image; `ubuntu-latest` moves to a new release on its own |
| One self-hosted runner, on the NAS, for the jobs that need the home network: plan and apply of `iac-home`, deployments of `app-home` (`runs-on: [self-hosted, nas]`) | the router and the NAS without a tunnel; everything else stays on GitHub ([security.md](https://github.com/bearclave/governance/blob/main/standards/security.md#workflows), `ci` `docs/runners.md`) |
| The NAS runner is the official image `ghcr.io/actions/actions-runner` pinned by digest, ephemeral, one job at a time | nothing to patch but the image, nothing kept between jobs; a job installs what the image lacks (the Azure CLI, pinned) |
| WireGuard from a GitHub runner is the fallback for `iac-home` | `terragrunt-home.yml` with `vpn-script`, when the NAS is down |
| A bump of the image is a `deps` pull request in `bearclave/ci` | all workflows move together and the plan shows the effect |
| The callers in the repositories run no job of their own | `runs-on` lives only in the shared workflows of `ci` |

**What a job may take from the image** - nothing else is assumed to be there:

| Tool | Used for |
|---|---|
| `bash`, coreutils, `curl`, `tar`, `unzip`, `jq`, `git` | scripts and downloads |
| `python3` (standard library only) | the conventions, schema and Sentinel lint scripts |
| `yq` | YAML to JSON for those scripts (the configs, the objects of collections) |
| `az` | sign-in by OIDC and Azure calls (through `azure/login`) |
| `gh` | pull request comments and labels |
| `node` / `npx` | the Static Web Apps CLI and site builds, at a pinned package version |

Anything else is downloaded at a pinned version and checked against its
sha256 before it runs.

## Tools and where their version lives

Two places, nothing else:

| File | Holds | Read by |
|---|---|---|
| `security.yml` and `pr.yml` of `bearclave/ci` (`env`) | gitleaks, actionlint, zizmor, shellcheck; markdownlint-cli2 (image version and digest): version and sha256 | the `security` and `pr` jobs of every repository |
| **`.github/tool-versions.env` of each repository** | the tools its own pipeline installs: Terraform, Terragrunt, TFLint, Trivy, ... with sha256 | the shared workflow it calls, from the caller's checkout |

A repository's plan therefore runs exactly the versions in its own
`tool-versions.env`, whatever release of `ci` it calls.

| Tool | Installed by | Used in |
|---|---|---|
| Terraform | release binary + sha256 (or `hashicorp/setup-terraform` with the pinned version) | `terragrunt*`, `terraform-modules` |
| Terragrunt | release binary + sha256 | `terragrunt*` |
| TFLint | release binary + sha256 | `terraform-modules` |
| Trivy | release binary + sha256 | `terragrunt*`, `terraform-modules` |
| gitleaks, actionlint, zizmor, shellcheck | release binary + sha256 | `security` |
| markdownlint-cli2 | container image pinned by digest | `pr` |
| Static Web Apps CLI | `npx` at a pinned version | `static-web-app` |
| ESLint, `@eslint/js`, `globals` | `npx` at a pinned version, or the repository's `package.json` | `static-web-app`, `container`, `checks` |

**One version across the organisation.** Every `tool-versions.env` that
pins a tool pins the same version: today **Terraform 1.16.4** and
**Terragrunt 1.1.6**. `deps.yml` bumps them in all repositories in the same
month; a repository that falls behind is a `Debt` issue.

- **Bumps:** Dependabot does not read `tool-versions.env`; `deps.yml` in
  `bearclave/ci` does, monthly (see [Dependency updates](#dependency-updates)).
- **New tool:** only when no existing tool or the image can do the job; it
  goes into `tool-versions.env` and the table above in the same pull request.

## Languages

| Language | Version | Where pinned |
|---|---|---|
| Python | the runner's `python3` (3.12 on `ubuntu-24.04`), standard library only in pipeline scripts | the runner image |
| Python of an application | `requires-python` in `pyproject.toml`, the base image in the `Dockerfile` | the repository |
| Node.js | the runner's `node` for tooling; an Azure Functions app pins its runtime in `staticwebapp.config.json`/`package.json` `engines` | the repository |
| TypeScript | `typescript` in `devDependencies` | `package.json` |
| PowerShell | the runner's `pwsh` | the runner image |
| HCL | Terraform and Terragrunt above | `tool-versions.env` |

## Terraform providers

| Where | What is pinned | Why |
|---|---|---|
| Modules (`terraform-modules`) | a lower bound only: `>= 5.0` | a module works with any newer provider the configuration chooses |
| Configuration (every Terragrunt repository) | the exact version, once, in `account.hcl` (`versions`), generated into every unit from `_common/templates/versions.tf.tmpl` | one place per repository; every unit runs the same provider |
| `.terraform.lock.hcl` | committed in every unit, with hashes for `linux_amd64` and `darwin_arm64` | the provider binary is verified, not only its version |

- Lock files are refreshed with
  `terragrunt run --all providers lock -platform=linux_amd64 -platform=darwin_arm64`
  in the same pull request that bumps a version.
- A provider bump is a `deps` pull request with its plan; a major version is
  its own pull request with the upgrade guide linked in Why.
- `required_version` of Terraform is a lower bound in modules and follows
  `tool-versions.env` in configuration.
- `.terraform.lock.hcl` is **not** in `.gitignore`. The pipeline's
  `init` runs with `-lockfile=readonly`: a unit whose lock file is missing or
  would change fails the validate job.

## Dependency updates

| What | Who keeps it current | How often |
|---|---|---|
| Actions in the shared workflows (`ci` `templates/`, its own workflows) | `deps.yml` in `ci`: new SHA and tag comment, then `tools/build.py` | weekly |
| Tools in `.github/tool-versions.env` of every repository and `templates/security.yml` of `ci` | `deps.yml` in `ci`: version and sha256 (the release asset's `digest`), one pull request per repository | monthly |
| Providers pinned in `account.hcl` | `deps.yml` in `ci` | monthly |
| Provider lower bounds of the modules (`*-modules`) | not bumped; `deps.yml` lists them next to the latest release in its run summary | monthly |
| `bearclave/ci@vX.Y.Z` pins | `ci-pins.yml` in `ci`, one pull request per pinned repository, only when a called workflow changed and the tag is signed | after each `ci` release (weekly at most) |
| npm, Python (uv), Docker of applications | Dependabot, one grouped pull request per ecosystem ([templates/dependabot.yml](https://github.com/bearclave/governance/blob/main/standards/templates/dependabot.yml)) | monthly |
| Go modules of `terraform-provider-scc` | Dependabot `gomod`, grouped, 7 days of cooldown | monthly |
| The NAS runner image (`app-nas` `deploy/runner/compose.yaml`) | Dependabot `docker-compose`, 14 days of cooldown; GitHub stops giving jobs to a runner 63 days after a newer release | monthly |
| The Azure CLI of the NAS runner (`ci` `templates/_fragments/self-hosted-tools.yml`) | by hand in `ci` | with the monthly tool bump |
| Security updates | Dependabot security updates (repository settings) | at once |
| Actions in the other repositories | none: they only call `bearclave/ci` and the managed `pr.yml` | - |

- **Cooldown.** Only releases at least 7 days old; a compromised release is
  usually pulled within days. `deps.yml` proposes versions within the same
  major and lists a newer major in the pull request: a major is a pull
  request of its own, by hand, with the upgrade guide linked in Why.
- **Who opens, who merges.** `deps.yml` and `ci-pins.yml` run as the GitHub
  App `bearclave-bot` and open one `deps` pull request per repository
  (branches `deps/tools`, `deps/actions`, `deps/ci`), labelled `type:deps`
  and assigned to the owner. Every dependency pull request, Dependabot's
  included, is approved and merged by the owner; nothing merges by itself.
  A bot's pull request still open on its branch is left alone (`deps.yml`)
  or moved to the newer release (`ci-pins.yml`).
- **`dependabot.yml`** exists only in a repository with npm, uv or Docker
  packages. `ci` has none: Dependabot bumped the generated workflows, not
  `templates/` (ci#12).
- **Cost.** About 3-5 pull requests a month instead of a weekly run in every
  repository; a pull request that only changes `.github/dependabot.yml`,
  `.github/release.yml`, `.claude/` or Markdown outside the units plans
  nothing in a Terragrunt repository.

## Locally

The Mac runs the same versions as the pipeline: `tenv` installs Terraform
and Terragrunt at the versions of the repository's `.github/tool-versions.env`
(`tenv tf install <version>`, `tenv tg install <version>`), so a local plan
does not run on a different Terraform. Homebrew installs only what the
pipeline does not use.

## Where it is documented

| What | Where |
|---|---|
| These rules | this file |
| Versions and sha256 | `.github/tool-versions.env` of each repository; the security tools in `security.yml` of `bearclave/ci` |
| Exact provider versions | `account.hcl` of each configuration repository |
| What a pipeline runs | its row in [pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md) and the README of `bearclave/ci` |

A README of a repository does not list tool versions; its Pipeline section
names the shared workflow and links here.
