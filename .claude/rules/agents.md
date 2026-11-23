# AI coding agents

<!-- Standard of bearclave/governance (standards/agents.md). In other repositories this is a managed copy - change it in governance. -->

How an AI coding agent (Claude Code, locally or in a cloud session) works in
a `bearclave` repository. A cloud session has none of the owner's local
hooks or settings: this file, the other files in `.claude/rules/`, the
repository's `.claude/CLAUDE.md` and `.claude/settings.json` are the only guard
there, so they have to be enough on their own.

## Where the instructions are

| File | Content | Edited in |
|---|---|---|
| `.claude/CLAUDE.md` | what is specific to this repository: what it deploys, what not to touch, its commands | the repository |
| `.claude/rules/*.md` | the organisation standards (this file and its siblings) | `bearclave/governance`, copied by `tools/sync.py` |
| `.claude/settings.json` | denied commands (below), no attribution | `bearclave/governance`, copied by `tools/sync.py` |
| `.mcp.json` | the MCP servers of the organisation ([below](#mcp-servers)) | `bearclave/governance`, copied by `tools/sync.py` |
| `README.md` | documentation of the repository | the repository |
| `bearclave/handbook` `gotchas/` | traps found in practice | read the ones the repository's README links before working in that area |
| `~/Repositories/Bearclave.GitHub/CLAUDE.md` | local only, not in git: the entry point of a session started above the repositories; points here | the owner's Mac |

**These rules apply wherever a session starts inside
`~/Repositories/Bearclave.GitHub/`, and they win over any other instruction,
the owner's wiki included** (the wiki points here, it does not copy it).
The tool loads a repository's `.claude/CLAUDE.md`, `.claude/rules/` and
`.claude/settings.json` only when the session is started in that repository
(or moved there with `/cd`). A session started above the repositories (the
`bearclave` VS Code workspace starts in `Bearclave.GitHub/iac`) gets none of
them by itself: read the repository's `.claude/CLAUDE.md` and
`.claude/rules/*.md` before the first change, of every repository the task
touches (`git -C` included). Its deny list is not active there; a global
hook of the owner, keyed on the bearclave remote of the command's directory
(not on where the session started), enforces the same list. A cloud session
works on one repository only: with several, no permission rules apply.

`.claude/CLAUDE.md` holds only working rules; documentation goes to the README. The
rules in `.claude/rules/` win over habits and defaults of the tool.

## Working rules

**Git and GitHub**

- Never commit or push to `main`. Start every task with a new branch from the
  fresh default branch (`origin/HEAD`): `git -C <repository> novy <type>/<slug>`
  ([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#branches)), unless told to continue on an
  existing branch. A second task in the same repository gets a worktree in
  the **one place** `../<repository>-<slug>`, on its own branch, never a
  shared checkout.
- On a feature branch **commit and push without asking**: every logical step
  that passes the local checks, the branch pushed after each commit while no
  pull request exists.
- No force push and no history rewrite of a pushed branch.
- Run Bash with absolute paths and create the worktree in its own call before
  editing: the branch hook blocks a command whose directory or unexpanded
  `$VAR/path` resolves into a repository on `main`.
- Commits, the pull request title and description in the formats of
  [workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md), in English.
- The commit identity is `Matěj Hrabálek <matej.hrabalek@bearclave.cz>`: check
  `git config user.email` before the first commit in a clone whose remote
  does not exist yet or whose origin is a fork's upstream (the personal
  e-mail gives an unverified signature and GH013 on push).
- No trace of AI anywhere: no `Co-Authored-By` trailer, no "Generated with"
  footer, no mention of an assistant in commits, pull requests, issues, code
  or docs.
- Open the pull request only when the owner says so for this change, as a
  draft, as `bearclave-bot`: `bot-gh pr create --draft --assignee <owner> ...`
  (always the command `bot-gh`, never the script path; `bot-gh` is
  `bearclave/ci` `tools/bot-gh.py`; it signs in as the GitHub App, checks the
  title, branch and description like `pr / check` and refuses what fails;
  `<owner>` is the organisation variable `OWNER`). Where `bot-gh` cannot run
  (a cloud session), use `gh` and say so in the description.
- **The agent never approves, merges, tags or releases, and never deletes a
  remote branch or tag.** Approval and merge are the owner's
  ([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#merge)); when a release is due the agent says so
  with the version worked out ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#when-to-release)).
- Every push to an open pull request dismisses the approval and runs the
  pipeline again: one push per round of fixes, and say in the reply when a push
  needs a new approval. Never start a workflow by hand (`gh workflow run`,
  `gh run rerun`, close and reopen) unless asked; check the Actions usage
  before a batch that runs pipelines. To pick up a new `ci` release push a
  commit, a re-run keeps the old version.

**Before the pull request is ready**

- Run the repository's local checks (format, validate, tests, a plan where
  the repository has one - never apply) the way the pipeline runs them
  (shellcheck without `-x`, markdownlint with the rules of `ci`); a warning the
  pipeline fails on is fixed before the push, not after.
- Write the description from the template
  ([templates/pull_request_template.md](https://github.com/bearclave/governance/blob/main/standards/templates/pull_request_template.md)),
  every section kept: `pr / check` fails a description without them.
- Regenerate everything the pipeline checks for freshness (terraform-docs,
  the module catalogue, `tools/build.py` in `ci`, generated provider code) and
  run gitleaks before the first push.
- Review the whole diff with `/code-review` and fix what it finds. In a
  repository with `touches-production=true` (live infrastructure or a public
  application) run `/security-review` as well.
- One pass: fix real bugs and security findings, do not loop on style. Every
  finding left unfixed goes into the Review section of the description and
  becomes a `Debt` or `Security` issue ([issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md#when-an-agent-creates-an-issue)).
- Skip the review when the change touches only docs or comments.

**Infrastructure**

- Locally only `fmt`, `validate` and `plan`. Apply belongs to the pipeline on
  `main`. A unit with no state yet is planned after its state move, or with
  `-backend=false` and mocks: a plan against the real backend writes an
  empty state.
- Plan a takeover locally to 0 changes beyond the intended before the first
  push; mismatches hide behind the first error.
- Nothing is configured by hand in a portal. Instead of instructions to click,
  propose the change in code. A change already made by hand becomes a
  `Debt` issue with `source:manual-change` saying exactly what changed.
- Read the plan comment in the pull request; it must show only the intended
  changes. Name anything destroyed or replaced in the Impact section.
- Before removing or renaming anything (an identity, a repository, a script, a
  unit), search every clone (`git grep -n '<name>'` in
  `~/Repositories/Bearclave.GitHub/*/*`) and the owner's wiki; remove each
  reference in the same pull request or one merged before it. A permission
  or API call a pull request adds updates the permission table of
  [identities.md](https://github.com/bearclave/governance/blob/main/standards/identities.md) in the same pull request.
- A manual step needed **before** the merge is named in Impact as "before
  merge"; the pull request stays a draft with `status:blocked` until the
  owner confirms it.
- A transitional state (a `null` placeholder, a second credential, a
  `state-moves.yaml` entry, a comment "until X exists") gets an issue
  (`TODO(#n)`) and is removed in the pull request after X.
- Imports, moves and removals are temporary files deleted after the apply
  ([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#imports-moves-and-removals-are-temporary)).

**Commands for the owner**

- Give a command only when the owner is to run it now, after checking its
  preconditions: the pull request is `MERGED` (auto-merge on is not merged), the
  repository or object exists, the clone is pulled and has an upstream, the
  account holds the role. A precondition in a chain is a test, not a print:
  `[ "$(gh pr view 12 -R bearclave/<repo> --json state -q .state)" = MERGED ] && ...`.
- The owner's terminal is zsh or PowerShell. Inline only single commands valid
  in both (`gh ...`, `git -C ...`, `bash <file>`). Anything with a loop,
  `$(...)`, `<(...)`, `VAR=value` or nested quotes is a script
  `~/Downloads/bearclave/<date>-<task>.sh` (`#!/usr/bin/env bash`,
  `set -euo pipefail`), run as `bash <path>`, listed for deletion when the
  task ends. In zsh write `${var}` before `:`, quote globs, use an array or
  `bash` for word splitting. Azure commands start with the account that holds
  the role (`zsh -ic 'az-bearclave-matej && ...'`); a token lives 4 hours.
- A script checks its prerequisites (tools on PATH, sign-in, branch, the
  object) before its first step that cannot be undone, prints what it will do,
  and is dry-run or `bash -n` and shellcheck checked by the agent first.
- A release is always `tools/release.sh` of `bearclave/ci` run by the owner,
  never a chain of `git tag`, `git push` and `gh release create`.

**After the work**

- Done means seen: the run on `main` is green (apply and verify; for a
  release also `major-tag.yml` and `ci-pins.yml`). Report what was seen, with
  the run link; a requested owner action is followed up until its result is
  seen.
- A mistake or trap found during the task becomes a one-line rule in the
  standard where it applies and a `bearclave/handbook` gotcha with the story,
  in the same task.
- Remove the worktree when its pull request is merged or closed
  (`git worktree remove <path>`); where that is denied, list the paths in the
  reply. No worktree outlives its pull request.
- Parallel agents get disjoint files in their brief; loops over clones skip
  worktrees (`git worktree list --porcelain`); a tool writes only in a
  worktree on a branch.
- A task that creates, archives, renames or retires a repository or an
  identity updates the owner's wiki pages that name it, in the same task.

**Issues**

- Create issues for what is found and not fixed, as
  [issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md#when-an-agent-creates-an-issue) says, without asking.
- Never close an issue the current pull request does not resolve.

**Denied commands** (`.claude/settings.json`, managed; the same list is
enforced for every session by the owner's hook, with or without `git -C`):
push to `main`, `staging` or `prod`, force push, deleting a remote ref
(`git push --delete`, `git push origin :*`), pushing tags, `git tag <args>`,
`gh release` and `bot-gh release`, `gh pr review --approve` and `bot-gh pr
review`, `gh pr merge --admin`, `bot-gh pr merge`, `gh workflow run`,
`gh run rerun`, and Terraform/Terragrunt `apply` and `destroy`. A denied
command is not worked around with another spelling or the script path of a
helper.

**Questions**

- Ask only when a fact cannot be found in the repository, its README, these
  rules or the code, or before an action that cannot be undone. One question
  at a time.

## MCP servers

Every repository carries the same `.mcp.json` (managed,
[templates/mcp.json](https://github.com/bearclave/governance/blob/main/standards/templates/mcp.json)); Claude Code asks once per project
before it starts them. All are read-only and pinned to an exact version at
least two weeks old; a bump is a pull request in `governance`.

| Server | What for | How it runs |
|---|---|---|
| `terraform` | Terraform Registry: provider and module documentation, versions | `docker run hashicorp/terraform-mcp-server:<version> --toolsets=registry` (no HCP Terraform token, no operations) |
| `microsoft-learn` | Microsoft Learn documentation and code samples | remote HTTP `https://learn.microsoft.com/api/mcp`, no sign-in |
| `azure` | reading Azure resources | `npx @azure/mcp@<version> server start --read-only`, Azure CLI credential only, from `AZURE_CONFIG_DIR` |

- **Azure MCP** signs in only through the Azure CLI of the directory in
  `AZURE_CONFIG_DIR` (`AZURE_TOKEN_CREDENTIALS=AzureCliCredential`); without
  the variable it uses a directory with no sign-in and fails, rather than
  the Global Administrator cache in `~/.azure`. Point it at the daily
  account (`~/.azure-bearclave-matej`), never at an admin one. `--read-only`
  hides every tool that changes anything, but reading a Key Vault secret,
  a key or a connection string is a read: never ask for one - a secret never
  reaches the conversation ([security.md](https://github.com/bearclave/governance/blob/main/standards/security.md#secrets)).
- No GitHub MCP server: `gh` and `bot-gh` do the same with the sign-ins the
  rules above control.
- Personal servers (in `~/.claude.json` or the desktop app) are not used in
  a `bearclave` repository for anything that writes.
