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
| `~/Repositories/Bearclave.GitHub/CLAUDE.md` | local only, not in git: the entry point of a session started in the folder above the repositories; points here | the owner's Mac |

Read the repository's `.claude/CLAUDE.md` before the first change. A
session that works on several repositories (`git -C`, scripts) reads the
`.claude/CLAUDE.md` of each it touches; Claude Code loads the rules of a
repository only once a file in it is read.

`.claude/CLAUDE.md` holds only working rules; documentation goes to the README. The
rules in `.claude/rules/` win over habits and defaults of the tool.

## Working rules

**Git and GitHub**

- Never commit or push to `main`. Start every task with a new branch from a
  fresh `origin/main` (`git novy <type>/<slug>`, see
  [workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#branches)), unless told to continue on an existing
  branch. Parallel tasks in one repository each get a git worktree
  (`git worktree add -b <type>/<slug> ../<repository>-<slug> origin/main`),
  never a shared checkout.
- No force push and no history rewrite of a pushed branch.
- Commits, the pull request title and description in the formats of
  [workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md), in English.
- No trace of AI anywhere: no `Co-Authored-By` trailer, no "Generated with"
  footer, no mention of an assistant in commits, pull requests, issues, code
  or docs. If the git identity in the environment is an anthropic.com
  address, commit as `Matěj Hrabálek <matej.hrabalek@bearclave.cz>`.
- Push the branch when the work is committed; open the pull request only
  when the owner says so for this change - every push to a pull request
  runs the pipeline and costs Actions minutes. Never start a workflow by
  hand (`gh workflow run`, a re-run) unless asked.
- Open the pull request as `bearclave-bot`: push the branch with git, then
  `bot-gh pr create --draft --assignee <owner> ...` (`bot-gh` is
  `bearclave/ci` `tools/bot-gh.py`; it signs in as the GitHub App and runs
  `gh`; `<owner>` is the organisation variable `OWNER`). The commits stay the
  owner's and signed; the bot only makes the owner able to approve. Where
  `bot-gh` cannot run (no Azure sign-in, a cloud session), open it with `gh`
  and say so in the description.
- Never approve a pull request - not as the bot, not with the owner's `gh`
  sign-in. The approval is the owner's own act (workflow.md#review).
- Do not merge a pull request unless the owner said so for this change; once
  approved, `bot-gh pr merge --squash` merges as the bot. Never `--admin`.
- Never tag, publish a release, or delete a remote branch or tag: say that a
  release is due ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#when-to-release)) and leave
  it to the owner.

**Before the pull request is ready**

- Run the repository's local checks (format, validate, tests, a plan where
  the repository has one - never apply).
- Review the whole diff with `/code-review` and fix what it finds. In a
  repository with `touches-production=true` (live infrastructure or a public
  application) run `/security-review` as well.
- One pass: fix real bugs and security findings, do not loop on style. Every
  finding left unfixed goes into the Review section of the description and
  becomes a `Debt` or `Security` issue ([issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md#when-an-agent-creates-an-issue)).
- Skip the review when the change touches only docs or comments.

**Infrastructure**

- Locally only `fmt`, `validate` and `plan`. Apply belongs to the pipeline on
  `main`.
- Nothing is configured by hand in a portal. Instead of instructions to click,
  propose the change in code. A change already made by hand becomes a
  `Debt` issue with `source:manual-change` saying exactly what changed.
- Read the plan comment in the pull request; it must show only the intended
  changes. Name anything destroyed or replaced in the Impact section.

**Issues**

- Create issues for what is found and not fixed, as
  [issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md#when-an-agent-creates-an-issue) says, without asking.
- Never close an issue the current pull request does not resolve.

**Denied commands** (`.claude/settings.json`, managed): push to `main`,
force push, deleting a remote ref (`git push --delete`, `git push origin :*`),
`git tag <args>`, `gh release create`, `gh pr review --approve`,
`gh pr merge --admin`, and Terraform/Terragrunt `apply` and `destroy`. A
denied command is not worked around with another spelling.

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
