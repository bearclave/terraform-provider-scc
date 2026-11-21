# Repositories

<!-- Standard of bearclave/governance (standards/repositories.md). In other repositories this is a managed copy - change it in governance. -->

## Groups and names

GitHub has no folders. A repository's group shows in four places that always
agree: the **name prefix**, the custom property **`group`**, a **topic** and
the organisation **Project** of the group. Locally the group is a folder in
`~/Repositories/Bearclave.GitHub/`.

| Group | Prefix | Project | Local folder | What belongs here |
|---|---|---|---|---|
| `iac` | `iac-` | IaC | `iac/` | configuration applied by a pipeline, and the modules it uses |
| `app` | `app-` | Apps | `app/` | applications and websites, and tools that build something |
| `template` | `template-`, `content-` | Templates & content | `template/` | what is copied to a customer: IaC templates, content catalogues |
| `platform` | none | Platform | `platform/` | how the organisation works: `ci`, `governance`, `handbook`, `ops-tools`, `.github`, `vs-code-extensions` |

Name: `<prefix><subject>`, lowercase, words joined by `-`, English, the
subject says what the repository manages (`iac-azure`, `app-web-bearclave`,
`content-sentinel`). No `-repo`, no `-new`, no year. Repositories of a
customer: [customers.md](https://github.com/bearclave/governance/blob/main/standards/customers.md).

One exception: a fork of a Terraform provider is named
`terraform-provider-<name>` and is public, because the Terraform Registry
publishes a provider only from such a repository
(`terraform-provider-scc`, group `app`).

An IaC repository of the Microsoft cloud is named after the product or
security domain it manages, in the singular, the way an administrator looks
for it: `iac-identity`, `iac-conditional-access`, `iac-intune`,
`iac-exchange`, `iac-purview`, `iac-sentinel`. The subject is also the
`<subject>` of its environment, identities and state container
(`intune-prod`, `sp-tf-intune`, `tfstate-intune`). A product it covers
alongside (Defender for Endpoint in `iac-intune`, Defender for Office 365 in
`iac-exchange`) does not get its own repository while it has one admin role
and one provider with it.

## Custom properties

Set on every repository in `bearclave/iac-github`; the rulesets target them.

| Property | Values | Drives |
|---|---|---|
| `group` | `iac`, `app`, `template`, `platform` | grouping, Project, topic |
| `pipeline` | `terragrunt`, `terragrunt-home`, `terraform-modules`, `static-web-app`, `container`, `content-sentinel`, `checks`, `ci`, `template`; `m365-dsc` only on `iac-m365` until it is archived | which shared workflow `ci.yml` calls, which checks are required |
| `entity` | `sro`, `osvc`, `shared` | whose the repository is (Bearclave s.r.o., the sole trader, both) |
| `touches-production` | `true`, `false` | branch up to date before merge, `/security-review` |
| `direct-commit` | `true`, `false` | `true` only for repositories written straight to `main` (no pull request) |

## What every repository contains

Where everything else goes - root files, `.github/`, the folder vocabulary,
`docs/` - is [layout.md](https://github.com/bearclave/governance/blob/main/standards/layout.md).

| Path | Owner | Content |
|---|---|---|
| `README.md` | repository | documentation, structure in [documentation.md](https://github.com/bearclave/governance/blob/main/standards/documentation.md) |
| `.claude/CLAUDE.md` | repository | working rules specific to the repository ([templates/CLAUDE.md](https://github.com/bearclave/governance/blob/main/standards/templates/CLAUDE.md)) |
| `.claude/settings.json` | managed | denied commands, no attribution |
| `.claude/rules/*.md` | managed | these standards: the always-rules and the language rules of the repository's languages |
| `.mcp.json` | managed | the MCP servers of the organisation ([agents.md](https://github.com/bearclave/governance/blob/main/standards/agents.md#mcp-servers)) |
| `.github/workflows/ci.yml` | repository | calls the shared workflow of its `pipeline` ([templates/workflows/](https://github.com/bearclave/governance/tree/main/standards/templates/workflows/)) |
| `.github/workflows/pr.yml` | managed | the `pr` check: title, branch, labels, assignee, root layout |
| `.github/workflows/drift.yml` | repository | only where something is applied: the weekly drift |
| `.github/CODEOWNERS` | managed (written when missing) | `* @bearclave/owners` ([access.md](https://github.com/bearclave/governance/blob/main/standards/access.md)) |
| `.github/SECURITY.md` | managed | how to report a weakness |
| `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md` | managed | the issue forms and the pull request description |
| `.github/markdownlint.yaml` | repository | only where a markdownlint rule differs from `ci`, each with a comment why ([documentation.md](https://github.com/bearclave/governance/blob/main/standards/documentation.md#markdown)) |
| `.github/dependabot.yml` | repository | only with npm, uv or Docker packages: from [templates/dependabot.yml](https://github.com/bearclave/governance/blob/main/standards/templates/dependabot.yml), the repository's ecosystems only |
| `.github/release.yml` | managed | release note categories |
| `.github/tool-versions.env` | repository | pinned tools with sha256, where the pipeline installs any ([toolchain.md](https://github.com/bearclave/governance/blob/main/standards/toolchain.md)) |
| `.github/conventions.yaml` | repository | conventions rules, layout exceptions - where there are any |
| `.editorconfig` | managed | indentation and line endings |
| `.gitattributes` | managed | `text=auto eol=lf`, binary formats; the repository's generated paths below the marker |
| `.gitignore` | repository | |
| `LICENSE` | repository | only where the code is handed over ([Metadata](#metadata)) |

What else a repository must have depends on its pipeline:
[repo-types.md](https://github.com/bearclave/governance/blob/main/standards/repo-types.md).

"Managed" files are written by `governance/tools/sync.py` and carry a header
saying so; a local edit is overwritten by the next sync (the section of
`.gitattributes` below its marker is kept, and `CODEOWNERS` is written only
when the repository has none). The pull request and issue templates are
copied too: GitHub reads the organisation defaults of `bearclave/.github`
only from a public repository, and that one is private. The `template-*`
repositories get only `.gitattributes`, without the header: they are copied
to customers and must not name the organisation.

## Where the settings live

Nothing about a repository is set by hand in the GitHub UI. Name,
description, topics, visibility, merge settings, custom properties,
environments, variables, labels, team access and rulesets are code in
[`bearclave/iac-github`](https://github.com/bearclave/iac-github); a change
there is a pull request like any other.

## Metadata

Set in `iac-github` for every repository; the `pr` check does not see it, the
plan of `iac-github` does.

| Item | Rule |
|---|---|
| Description | one sentence, at most 200 characters, what it manages and for whom |
| Topics | the group (`iac`, `app`, `template`, `platform`) and two or three subjects (`azure`, `terragrunt`, `website`) |
| Homepage | the URL of what it deploys (`https://bearclave.cz` for `app-web-bearclave`); empty otherwise |
| `CODEOWNERS` | `.github/CODEOWNERS` with `* @bearclave/owners` (managed) |
| `SECURITY.md` | `.github/SECURITY.md` (managed): report privately to the owner, never in an issue of a public repository |
| `LICENSE` | **internal code: none** - private repositories are proprietary to Bearclave s.r.o. by default; the README does not claim otherwise. **Code handed to a customer** (templates, a handover): an explicit `LICENSE` agreed in the contract. **A fork** keeps the upstream licence (`vs-code-extensions/extensions/pipeline-runner`: GPL-3.0, and its `package.json` says so) |

## Licensing

What the organisation pays for and on which plan is kept in Notion, in
**Systémy a přístupy** (Bearclave workspace), not here: GitHub (Team, seats,
Actions minutes and budget), Microsoft 365 and Entra licences, Azure
subscriptions, Claude (plan and seats), and the other tools. A change of a
plan is recorded there first; a standard that depends on a plan (GitHub Team
features in [docs/organisation-settings.md](https://github.com/bearclave/governance/blob/main/docs/organisation-settings.md))
links there.

## A new repository

1. Add it to `iac-github` (`infrastructure/repositories/<name>/terragrunt.hcl`) with its
   group, pipeline, properties, metadata, team access and - where it applies
   or deploys - its environment and variables; the merge creates it with
   labels, rulesets and settings.
2. Identities, where its pipeline signs in: a pull request in
   `iac-identity` (applications, credentials, permissions) and in
   `iac-azure` (state container and roles) first
   ([identities.md](https://github.com/bearclave/governance/blob/main/standards/identities.md#where-they-are-declared)), then the
   variables in `iac-github`.
3. Clone it into the folder of its group, `git novy feat/skeleton`.
4. Run `governance/tools/sync.py <name>` for it, add `README.md`,
   `.claude/CLAUDE.md`, `.github/workflows/ci.yml` (and `dependabot.yml`
   when it has npm, uv or Docker packages) from the templates, and what
   [repo-types.md](https://github.com/bearclave/governance/blob/main/standards/repo-types.md) asks for its pipeline.
5. Open the first pull request; it must already pass `security / scan`,
   `pr / check` and `pipeline / gate`.
6. Link it to the Project of its group (`gh project link`,
   [projects.md](https://github.com/bearclave/governance/blob/main/standards/projects.md)); `project-sync` in `ci` adds its issues and
   pull requests from then on.
7. Add it to the table in `work/repos.md` of the LLM wiki.

## Archiving

A repository nobody works on any more is archived (`archived = true` in
`iac-github`), not deleted. It stays in its Project and keeps its history.
