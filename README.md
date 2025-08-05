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

Initialize Claude Code hooks configuration with blocc.

```bash
# Initialize with interactive setup
$ blocc --init
Include stdout in error output? (y/N): y
Add stdout filter? (y/N): n
Add stderr filter? (y/N): n
Exclude stderr from error output? (y/N): n
Enter commands to run (one per line, empty line to finish):
make lint
make test

Successfully created .claude/settings.local.json
```

This creates `./.claude/settings.local.json`.

> **Note**: It's recommended to configure hooks to trigger on `Stop` events. Using `PostToolUse` hooks may cause the AI model to become distracted or consume extra context unnecessarily.

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "blocc --stdout 'make lint' 'make test'"
          }
        ]
      }
    ]
  }
}
```

## Usage

```bash
$ blocc --help
Usage: main [<commands> ...] [flags]

Arguments:
  [<commands> ...]    Commands to execute

Flags:
  -h, --help                    Show context-sensitive help.
  -v, --version                 Show version information
  -p, --parallel                Execute commands in parallel
  -m, --message=STRING          Custom error message
  -i, --init                    Initialize settings.local.json
  -s, --stdout                  Include stdout in error output
  -o, --stdout-filter=STRING    Filter command for stdout
  -e, --stderr-filter=STRING    Filter command for stderr
  -n, --no-stderr               Exclude stderr from error output

# Execute commands sequentially (default).
$ blocc "npm run lint" "npm run test"
{
  "message": "2 command(s) failed",
  "results": [
    {
      "command": "npm run lint",
      "exitCode": 1,
      "stderr": "Linting errors found..."
    },
    {
      "command": "npm run test",
      "exitCode": 1,
      "stderr": "Test failures..."
    }
  ]
}

# Execute commands in parallel(-p).
$ blocc --parallel "npm run lint" "npm run test" "npm run spell-check"

# Custom error message(-m).
$ blocc --message "Hook execution completed with errors. Please address the following issues" "npm run lint" "npm run test"

# Include stdout in error output(-s).
$ blocc --stdout "npm run lint" "npm run test"

# Filter output for context engineering(-o/-e).
$ blocc -n -s "cspell lint . --cache --gitignore" -o "perl -nle 'print \$1 if /Unknown word \((\w+)\)/' | sort | uniq"
{
  "message": "1 command(s) failed",
  "results": [
    {
      "command": "cspell lint . --cache --gitignore",
      "exitCode": 1,
      "stdout": "alecthomas\nBINPATH\nblocc\nBlocc\nclippy\nDISTPATH\ngofmt\ngolangci\nGOPATH\ngoreleaser\ngotextdiff\nhexops\nnonexistentcommand\nnosec\noicd\nprintln\nrepr\nshuntaka\nvxeg\n"
    }
  ]
}
````
