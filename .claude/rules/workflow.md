# Workflow: from the first line to `main`

<!-- Standard of bearclave/governance (standards/workflow.md). In other repositories this is a managed copy - change it in governance. -->

One task = one branch from a fresh `main` = one pull request = one squash
commit on `main`. Nobody pushes to `main`; the organisation rulesets in
`bearclave/iac-github` refuse it, for the owner too. The only exception are
repositories with `direct-commit=true` (`knowhow`, `office-templates`),
which the owner writes straight to `main`; agents work there on a branch
and a pull request as everywhere else.

**Applications** (`app-web-bearclave`, `app-web-matejhrabalek`, `app-home`)
have the default branch `staging` instead of `main`, and a second branch
`prod` ([below](#applications-staging-and-prod)). Wherever this standard says
`main`, an application reads its default branch, `staging`.

## Branches

```text
<type>/<slug>
<type>/<issue>-<slug>
```

- `type` is one of the commit types below (`feat`, `fix`, `refactor`, ...).
- `slug`: lowercase `a-z`, `0-9` and `-`, 2-5 words, says what the branch does:
  `fix/dcr-syslog-priority`, `feat/42-ca-block-legacy-auth`.
- The issue number goes first when the branch resolves an issue.
- Always from the fresh default branch (`origin/HEAD`: `main`, `staging` in an
  application): `git -C <repository> novy <type>/<slug>` fetches and branches
  in one step. The branch reaches GitHub with its first `git push`; nobody
  creates a branch through the GitHub UI or API for an agent's work. After a
  change of the default branch, `git remote set-head origin -a` in every clone.
- A second task in the same repository gets a worktree, never a shared
  checkout, in the one place `../<repository>-<slug>` on the branch
  `<type>/<slug>` (`git -C <repository> worktree add -b <type>/<slug>
  ../<repository>-<slug> origin/main`). It is removed after the merge
  ([housekeeping](#housekeeping)).
- Short-lived: merged or closed within days. A branch that waits on something
  stays a draft PR with the label `status:blocked` and a comment why.
- Bot branches keep their own names: `dependabot/...`; `bearclave-bot`'s `deps/tools`, `deps/actions` and
  `deps/ci` already have the `<type>/<slug>` form.
- The `pr` check refuses any other branch name.

**How it is enforced.** The branch name and the title are checked by the
required check `pr / check`, not by a ruleset: the ruleset rules for branch
names and commit messages exist only in GitHub Enterprise. Push rulesets on
public repositories and the merge queue are Enterprise-only too, so the `pr`
check and the settings below stand in for them. A branch with a wrong name can
be pushed, but its pull request cannot be merged. The commits
on a branch are not checked; only the title reaches `main`.

## Commits

[Conventional Commits 1.0](https://www.conventionalcommits.org/en/v1.0.0/),
template in [templates/commit-message.txt](https://github.com/bearclave/governance/blob/main/standards/templates/commit-message.txt):

```text
<type>(<scope>)!: <summary>

<body: why the change is needed, what it does not do, wrapped at 72>

Refs #12
BREAKING CHANGE: <what callers must change>
```

| Type | Use for | Release effect |
|---|---|---|
| `feat` | new behaviour, resource, setting, page | minor |
| `fix` | wrong behaviour corrected, drift fixed | patch |
| `perf` | faster or cheaper, same behaviour | patch |
| `refactor` | structure changed, behaviour and plan unchanged | patch |
| `docs` | documentation and comments only | none |
| `test` | tests only | none |
| `ci` | workflows, pipeline scripts, repository tooling | none |
| `chore` | anything else that changes no behaviour (housekeeping, generated files) | none |
| `deps` | dependency, provider, action or tool version bump | patch |
| `revert` | reverts an earlier commit, names it in the body | patch |

- **Scope** is the area the change lives in: a module (`entra-groups`), a unit
  or component (`sentinel`, `rg-tfstate`), a workload (`exchange`), a part of
  the app (`api`). Lowercase, one word or a hyphenated term. Optional, but
  use it whenever the repository has more than one area.
- **Summary**: imperative, lowercase first letter, no full stop, at most 72
  characters for the whole first line: `fix(sentinel): read the priority of
  DSM syslog messages`.
- **`!`** and a `BREAKING CHANGE:` footer when a consumer has to change
  (a module input renamed, a reusable workflow input removed).
- **Body** explains *why*, not *what* - the diff shows what. Required unless
  the summary says everything.
- **Footer**: `Refs #12` for a related issue, `Closes #12` only in the pull
  request description (see below).
- The commit on `main` is the squash commit GitHub signs. Branch commits are
  signed with the owner's key wherever it is available (not in a cloud
  session), with the identity verified on GitHub (`@bearclave.cz`).
- No trace of AI anywhere: no `Co-Authored-By`, no "Generated with", no
  mention of an assistant.

**How often.** Commit every logical step that passes the local checks of the
repository (format, validate, tests), without asking. Before the pull request
exists, push after every commit: a push to a branch without a pull request
runs nothing, and an unpushed branch is one nobody else can see and nothing
backs up. Once the pull request is open, push only a finished batch that
deserves a new full run; never push or edit the description just to re-run a
check. Commits on a branch may be rough - only the pull request title reaches
`main` - but they follow the format as well.

## Pull requests

**Open the pull request when the change is ready for its first full run,
and for an agent only when the owner says so for that change, as a draft.**
A pushed branch without a pull request costs nothing; every push to a pull
request and every edit of its title or description runs the pipeline and
costs Actions minutes (3,000 a month for the whole organisation): write the
description once, completely. When the pipeline is green and the Review
section is filled, the author marks it ready (`bot-gh pr ready <number>`);
the owner approves.

**Title** = the future commit on `main`, in the commit format above:
`feat(conditional-access): block legacy authentication`. The `pr` check
refuses a title that does not match, and sets the labels from it. A title
of a bot (Dependabot, `bearclave-bot`) longer than 72 characters only warns,
because the bot writes it; shorten it before the merge.

**Description** from [templates/pull_request_template.md](https://github.com/bearclave/governance/blob/main/standards/templates/pull_request_template.md)
(`.github/pull_request_template.md`, managed; GitHub fills it in). The
sections are not optional, for `bearclave-bot` too; write "none" rather than
delete one. Only Dependabot's pull requests are exempt:

| Section | Content |
|---|---|
| Why | the problem or goal, in one to three sentences |
| What changes | bullet list of the changes a reviewer should know about |
| Impact | production yes/no, what the plan or deployment does, anything destroyed or replaced, manual steps before merge and after merge |
| Verification | what was run and seen: plan summary, tests, screenshot, run link |
| Review | summary of the `/code-review` (and `/security-review`) pass and what was left unfixed - each left-over finding becomes an issue |
| Links | `Closes #12` for the issue the PR resolves, `Refs #` for related ones |

**Size.** One logical change per pull request. The `pr` check labels the
size from the changed lines, generated files and lock files excluded:

| Label | Changed lines | Expectation |
|---|---|---|
| `size:xs` | < 10 | |
| `size:s` | < 50 | |
| `size:m` | < 200 | the normal case |
| `size:l` | < 500 | explain in Why why it cannot be split |
| `size:xl` | >= 500 | split it, unless the change is mechanical or generated |

Split along natural seams: a module change and its use in a unit are two
pull requests (module first); a rename and a behaviour change are two; a new
app feature and its refactoring are two.

**Labels.** The `pr` check sets `type:*`, `breaking` and `size:*`. Add by
hand where it applies: `prio:high`, `status:blocked`. Full list in
[issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md#labels).

**Assignee** is the author; the `pr` check assigns them when nobody is.
Dependabot pull requests are assigned to the owner by `dependabot.yml`,
`bearclave-bot`'s `deps` pull requests by the workflow that opens them.

**Author.** A pull request an agent prepares is opened by the GitHub App
`bearclave-bot` (`bot-gh pr create ...`, the helper
[`bearclave/ci` `tools/bot-gh.py`](https://github.com/bearclave/ci/blob/main/tools/bot-gh.py)),
so that the owner can approve it: nobody can approve their own pull request.
The commits stay the owner's, signed with his key; only the author of the
pull request is the bot.

## Review

1. The author reviews the whole diff before marking the PR ready. An AI agent
   runs `/code-review`, and `/security-review` in repositories with
   `touches-production=true`; one pass, real bugs and security findings fixed,
   the rest written into the Review section and filed as `Debt` issues
   (`source:review`).
2. All required checks green: `security / scan`, `pr / check`,
   `pipeline / gate` (names explained in [pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md)).
3. For infrastructure: the plan comment in the pull request shows only the
   intended changes. Anything destroyed or replaced is named in Impact.
4. Every review conversation resolved (the ruleset requires it).
5. **The owner approves every pull request.** The ruleset `main-pull-request`
   requires one approving review and dismisses it on every new push, so the
   approval is always of the final diff. Pull requests of `bearclave-bot` and
   of Dependabot are approved the same way. A pull request the owner opened
   under his own account cannot be approved; he merges it through the
   ruleset's bypass (organisation owners, pull requests only). GitHub lets
   the bypass merge any pull request; by this rule the owner uses it only
   for his own. An agent never approves a pull request, under any account,
   and never merges with `--admin`.
6. **Destroys and replaces need an approval given after the plan.** In a
   Terragrunt repository the check `destroy / approved` (part of
   `pipeline / gate`) passes only when the pull request was approved after
   its latest plan; approving re-runs it. The push of the merge then applies
   the destroys by itself ([pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md#rules-for-every-workflow)).

## Merge

- **Squash merge only.** The commit on `main` takes the pull request title
  and description (repository setting), so both have to be final before
  merging.
- **Approve, then auto-merge.** Once the review is done, the owner approves
  and turns on auto-merge (`gh pr merge --auto --squash`, or the button); he
  runs "Update branch" right before approving.
  GitHub merges as soon as the required checks and the approval are there,
  and nobody waits for the last check by hand. A new push dismisses the
  approval, so auto-merge waits for the next one. Auto-merge does not use the
  bypass: a pull request the owner opened himself he merges directly.
- **No merge queue.** GitHub offers it for private repositories only on
  Enterprise Cloud, and with one owner merging a few pull requests a day it
  would add nothing: "Branch up to date with `main`" below keeps the merges
  of production repositories in order.
- **Branch up to date with `main`** in repositories that touch production
  (the ruleset requires it): "Update branch" re-runs the plan on the merged
  state.
- **Who merges: the owner, always.** Every pull request, dependency pull
  requests (Dependabot, `bearclave-bot`'s `deps`) included
  ([toolchain.md](https://github.com/bearclave/governance/blob/main/standards/toolchain.md#dependency-updates)). An agent never approves,
  merges, tags or releases, and never uses `--admin`
  ([agents.md](https://github.com/bearclave/governance/blob/main/standards/agents.md#working-rules)).
- The branch is deleted by GitHub after the merge. Locally `git uklid` removes
  the branch and `git worktree remove <path>` the worktree, the same day;
  `git-sync-main` (which follows the default branch) removes the clean
  worktrees of merged branches daily.
- **After the merge**, check the run on `main`: the apply or deployment went
  through, or the run is red and says why. A red run on `main` is fixed the
  same day.
- Revert with `git revert` in a new pull request titled
  `revert: <original title>`; never rewrite `main`. In an application this
  is the rollback ([below](#applications-staging-and-prod)).

## Applications: staging and prod

An application runs in two environments, each from its own branch; the IaC
repositories stay on `main`.

| Branch | Runs in | Moved by |
|---|---|---|
| `staging` (default) | staging: a Static Web App of its own on its `*.azurestaticapps.net` name with `X-Robots-Tag: noindex`, or the staging container | squash merges of pull requests, as on `main` elsewhere |
| `prod` | production (the custom domain, the NAS) | only `promote.yml`, as the App `bearclave-promote`, only by a fast-forward to a commit of `staging` |
| `<type>/<slug>` | a website's pull request: a preview environment of the staging SWA (`pr<number>`, noindex) | the author |

No other long-lived branch. Nobody pushes to `prod`, the owner included: the
ruleset on `prod` lets only the App `bearclave-promote` create or move it
(no owner bypass), never by a force push, and no pull request targets it.
`bearclave-promote` is installed only on the applications, and its key is
read only by their `promote` environment; `bearclave-bot` has nothing to do
with `prod`. No tag may be called `staging` or `prod`.

1. **Develop** as everywhere: `git -C <repository> novy <type>/<slug>` (from `origin/staging`),
   a pull request into `staging`, review, squash merge.
2. **Staging.** The merge builds the commit once, deploys it to staging and
   runs the smoke test. A red staging run is fixed the same day; such a
   commit cannot be promoted.
3. **Release by a click.** The owner runs `promote` (Actions, workflow
   `promote`, Run workflow on `staging`; `dry-run` first if in doubt). It
   checks that the staging run of the commit succeeded and deployed
   staging, and fast-forwards `prod` to it.
4. **Production.** The push to `prod` deploys the same build that staging ran
   (a website: the artifact of that staging run; a container: the image of
   that commit), never a rebuild, and runs the smoke test. `promote` waits
   for that run and only then tags `vX.Y.Z` and publishes the release
   ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#applications)): a release is what
   production runs.

**Rollback** is forward only: a `revert: <original title>` pull request into
`staging`, its staging run, then `promote` (a patch release). `prod` never
moves back and no older build is redeployed by hand, so `prod` is always
what runs. Whatever else was merged into `staging` since goes along with
the revert; revert it too, or hold its merge until the rollback is out. A
production deployment that failed for a transient reason is a manual run of
`ci` on `prod`: it deploys the same build again.

## Housekeeping

Branches and pull requests do not linger; nothing is deleted automatically,
since a deleted branch or a closed pull request is easy to lose track of.

| What | Stale after | What happens |
|---|---|---|
| Branch without a pull request | 30 days | open a draft pull request for it or delete it |
| Pull request (draft or ready) without any update | 14 days | finish it, mark it `status:blocked` with a comment why, or close it |
| Merged branch | at once | GitHub deletes it (repository setting); `git uklid` removes the local branch and its worktree |

A weekly report (`housekeeping` of `bearclave/ci`, one run for the whole
organisation) lists what is stale in one issue per repository, assigned to
the owner, and updates that issue instead of opening a new one; pull
requests labelled `status:blocked` are listed apart. Tags are never cleaned up this way: release tags are never
deleted ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#tags)).

## Notifications

GitHub's own notifications, read by e-mail and in the GitHub app; no
webhook or bot posts to Teams.

| The owner gets | Because |
|---|---|
| a review request or an assignment of a pull request | `CODEOWNERS` requests the review of `@bearclave/owners`; the `pr` check, `dependabot.yml` and the `deps` workflows of `ci` assign the owner |
| an issue assigned to him | issues are assigned when they are created ([issues.md](https://github.com/bearclave/governance/blob/main/standards/issues.md)) |
| a failed workflow run he started, or of a schedule he last changed | GitHub notifies the actor of the run |
| Dependabot alerts | weekly e-mail digest |
| the housekeeping issue | it is assigned to him |

Settings of the owner's account (Settings → Notifications):

- Watching: "Participating and @mentions"; no notifications of releases.
- E-mail for: review requests, assignments, failed workflows; not for
  every comment on a watched repository.
- Dependabot alerts: e-mail, weekly digest.

The owner's morning briefing reads the same through `gh`: pull requests
waiting for his approval, red runs on `main`, open issues labelled
`source:drift`, `prio:high` or of the type `Security`. A run on a schedule
notifies only whoever changed its cron last, so the briefing is what catches
a red drift run.

## Where the rules live

| What | Where |
|---|---|
| Branch protection, merge settings, required checks, labels, issue types | [`bearclave/iac-github`](https://github.com/bearclave/iac-github) |
| PR and issue templates, CODEOWNERS | `governance` `standards/templates/`, copied into every repository by `tools/sync.py` |
| Shared workflows and their checks | [`bearclave/ci`](https://github.com/bearclave/ci) |
| These rules | [`bearclave/governance`](https://github.com/bearclave/governance) `standards/` |
