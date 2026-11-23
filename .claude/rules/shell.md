---
paths:
  - "**/*.sh"
---

# Bash

<!-- Standard of bearclave/governance (standards/code/shell.md). In other repositories this is a managed copy - change it in governance. -->

- `#!/usr/bin/env bash` and `set -euo pipefail` on the second line; a
  one-line header comment saying what the script does, usage in the README.
- Comments: [documentation.md](https://github.com/bearclave/governance/blob/main/standards/documentation.md#comments-in-code).
- Quote every expansion (`"$var"`, `"${arr[@]}"`); `[[ ]]` or `[ ]`
  consistently; `$(...)`, never backticks.
- Functions for anything used twice; `local` variables inside them.
- Temporary files with `mktemp`, removed by a `trap ... EXIT`.
- Downloads are checked against a sha256 before use.
- No secret on a command line or in `set -x` output; read secrets into a
  variable, mask them in a workflow (`::add-mask::`).
- shellcheck clean at `--severity=warning` (the `security / scan` check runs
  it over every `*.sh` and every `run:` block).
- A script longer than ~150 lines or with real logic becomes Python.
- Commands typed in a terminal and the agent's Bash run in zsh: an unquoted
  `$var` is not split (`set -- $p` sets one argument), `$var:a` is a modifier
  (write `${var}:`), an unmatched glob is an error. Loops go into a `bash`
  script.
