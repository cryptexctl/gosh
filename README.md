# gosh - Go Shell

A POSIX-compatible shell written from scratch in Go.

## Why gosh?

* **Single static binary** – easy to distribute, cross-compile, no libc deps.
* **Memory-safe Go** – no UB, no legacy C code.
* **Fast startup** – <5 ms cold start on modern CPUs.
* **Built-in history with arrows** – ↑ / ↓ navigation like modern shells.
* **UTF-8 first class** – proper multibyte input handling.
* **0BSD license** – public-domain-like freedom.
* **Extensible** – drop-in Go plugins or built-ins.
* **Tiny codebase** – ~3 k LOC vs 150 k+ in GNU bash.

## Features

- Interactive shell with command history (arrow keys)  
- Bash-like control flow: `if/elif/else`, `while`, `for`  
- Logical operators `&&` `||`  
- Pipes & redirections (`|`, `>`, `<`, `>>`)  
- Background jobs (`&`) + `jobs/fg/bg/kill`  
- Variable substitution + inline assignments `VAR=1 cmd`  
- Minimal arithmetic `$((a+1))` and `[ 1 -lt 2 ]`  
- Script execution  
- Built-ins: `cd, pwd, echo, exit, help …`  
- UTF-8 input/output  
- Easter-eggs (compile with `-tags noeaster` to disable)

## Install

```bash
go install github.com/cryptexctl/gosh@latest
```

## Usage

```bash
gosh                 # interactive mode
gosh -c "echo hi"    # run one command
gosh script.sh       # run script file
```

## License

0BSD – do whatever you want. 