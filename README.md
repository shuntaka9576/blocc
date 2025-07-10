# blocc

`blocc` is a CLI tool that executes multiple commands and blocks Claude Code hooks by returning exit code 2 when any command fails.

1. blocc executed from Hooks caught errors from the specified command, consolidated output to stderr, and returned exit code 2 to provide feedback to Claude Code.
2. Claude Code automatically fixed the clippy warnings by updating the println! syntax to the modern format. ✨

![img](./docs/blocc-behavior.png)

## Installation

```bash
brew install shuntaka9576/tap/blocc
```

<details>
<summary>Go install</summary>

```bash
go install github.com/shuntaka9576/blocc/cmd/blocc@latest
```
</details>

<details>
<summary>From source</summary>

```bash
git clone https://github.com/shuntaka9576/blocc.git
cd blocc
make install
```
</details>

## QuickStart

Add blocc to your Claude Code hooks configuration in `~/.claude/settings.json` (global), `.claude/settings.json` (project), or `.claude/settings.local.json` (local).

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "blocc -p -m 'Hook execution completed with errors. Please address the following issues' 'pnpm lint' 'pnpm type-check' 'pnpm spell-check'"
          }
        ]
      }
    ]
  }
}
```

## Usage

```bash
# Execute commands sequentially (default).
blocc "npm run lint" "npm run test"

# Execute commands in parallel(-p).
blocc --parallel "npm run lint" "npm run test" "npm run spell-check"

# Custom error message(-m).
blocc --message "Hook execution completed with errors. Please address the following issues" "npm run lint" "npm run test"
```

When commands fail, blocc outputs a JSON structure to stderr. The output always includes both stdout and stderr from failed commands.

```bash
$ blocc "npm run lint" "npm run test"

{
  "message": "2 command(s) failed",
  "results": [
    {
      "command": "npm run lint",
      "exitCode": 1,
      "stderr": "Linting errors found...",
      "stdout": "Checking 15 files..."
    },
    {
      "command": "npm run test",
      "exitCode": 1,
      "stderr": "Test failures...",
      "stdout": "Running test suite..."
    }
  ]
}
```

This is particularly useful for tools like `cspell` that output actual error details to stdout.

```bash
$ blocc "cspell lint . --cache --gitignore"

{
  "message": "1 command(s) failed",
  "results": [
    {
      "command": "cspell lint . --cache --gitignore",
      "exitCode": 1,
      "stderr": "CSpell: Files checked: 39, Issues found: 2 in 1 file.",
      "stdout": "packages/iac/lib/deploy-role-stack.ts:15:11 - Unknown word (oicd)"
    }
  ]
}
```
