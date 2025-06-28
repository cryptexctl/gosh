# gosh - Go Shell

A POSIX-compatible shell implementation written from scratch in Go.

## Features

- Interactive shell with command history
- Script execution
- Pipes and redirections (`|`, `>`, `<`, `>>`)
- Background jobs (`&`)
- Variable substitution (`$VAR`, `${VAR}`)
- Job control (`jobs`, `fg`, `bg`, `kill`)
- Built-in commands (`cd`, `pwd`, `echo`, `exit`, `help`, etc.)

## Install

```bash
git clone <repository>
cd gosh
go build -o gosh .
```

## Usage

```bash
gosh                    # Interactive mode
gosh -c "command"       # Execute command
gosh script.sh          # Run script
gosh --help            # Show help
```

## Built-in Commands

- `cd [dir]` - Change directory
- `pwd` - Print working directory  
- `echo [args]` - Print arguments
- `exit [code]` - Exit shell
- `help [cmd]` - Show help
- `history` - Command history
- `export [var=value]` - Export variables
- `unset [var]` - Remove variables
- `jobs` - List jobs
- `fg [job]` - Foreground job
- `bg [job]` - Background job
- `kill [job]` - Kill job

## Examples

```bash
$ gosh
gosh 1.0.0 - Go Shell
user@host:~$ echo "Hello, World!"
Hello, World!
user@host:~$ export NAME="Go"
user@host:~$ echo "Hello, $NAME!"
Hello, Go!
user@host:~$ ls | head -3
user@host:~$ sleep 10 &
user@host:~$ jobs
user@host:~$ exit
```

## License

0BSD - Public Domain 