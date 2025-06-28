package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (s *Shell) builtinExit(args []string) int {
	code := 0
	if len(args) > 0 {
		if c, err := strconv.Atoi(args[0]); err == nil {
			code = c
		}
	}
	s.Exit(code)
	return code
}

func (s *Shell) builtinCD(args []string) int {
	var dir string

	if len(args) == 0 {
		dir = os.Getenv("HOME")
		if dir == "" {
			fmt.Fprintf(os.Stderr, "cd: HOME not set\n")
			return 1
		}
	} else {
		dir = args[0]
	}

	if dir == "-" {
		prevDir := s.variables.Get("OLDPWD")
		if prevDir == "" {
			fmt.Fprintf(os.Stderr, "cd: OLDPWD not set\n")
			return 1
		}
		dir = prevDir
		fmt.Println(dir)
	}

	if strings.HasPrefix(dir, "~") {
		home := os.Getenv("HOME")
		if home != "" {
			dir = filepath.Join(home, dir[1:])
		}
	}

	oldPwd, _ := os.Getwd()

	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		return 1
	}

	newPwd, _ := os.Getwd()
	s.variables.Set("OLDPWD", oldPwd)
	s.variables.Set("PWD", newPwd)
	s.currentDir = newPwd

	return 0
}

func (s *Shell) builtinPWD(args []string) int {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		return 1
	}
	fmt.Println(pwd)
	return 0
}

func (s *Shell) builtinEcho(args []string) int {
	output := strings.Join(args, " ")
	fmt.Println(output)
	return 0
}

func (s *Shell) builtinHelp(args []string) int {
	if len(args) == 0 {
		fmt.Println("gosh - Go Shell")
		fmt.Println()
		fmt.Println("Builtin commands:")

		builtins := []string{
			"cd [dir]      - Change directory",
			"pwd           - Print working directory",
			"echo [args]   - Print arguments",
			"exit [code]   - Exit shell",
			"help [cmd]    - Show help",
			"history       - Show command history",
			"export [var]  - Export variable",
			"unset [var]   - Unset variable",
			"set           - Show/set shell options",
			"source [file] - Execute file",
			". [file]      - Execute file (alias for source)",
			"jobs          - Show active jobs",
			"fg [job]      - Bring job to foreground",
			"bg [job]      - Send job to background",
			"kill [job]    - Kill job",
		}

		for _, builtin := range builtins {
			fmt.Printf("  %s\n", builtin)
		}

		fmt.Println()
		fmt.Println("For help on external commands, use 'man <command>'")
		return 0
	}

	cmd := args[0]
	switch cmd {
	case "cd":
		fmt.Println("cd [directory] - Change the current directory")
		fmt.Println("  cd           - Go to home directory")
		fmt.Println("  cd -         - Go to previous directory")
		fmt.Println("  cd /path     - Go to specified path")
	case "pwd":
		fmt.Println("pwd - Print the current working directory")
	case "echo":
		fmt.Println("echo [arguments...] - Display arguments")
	case "exit":
		fmt.Println("exit [code] - Exit the shell with optional exit code")
	case "history":
		fmt.Println("history - Display command history")
	case "export":
		fmt.Println("export [name[=value]] - Export variables to environment")
	case "unset":
		fmt.Println("unset [name] - Remove variable")
	default:
		fmt.Printf("No help available for '%s'\n", cmd)
		return 1
	}

	return 0
}

func (s *Shell) builtinHistory(args []string) int {
	if len(args) > 0 && args[0] == "-c" {
		s.history.Clear()
		return 0
	}

	entries := s.history.All()
	for i, entry := range entries {
		fmt.Printf("%4d  %s\n", i+1, entry)
	}

	return 0
}

func (s *Shell) builtinExport(args []string) int {
	if len(args) == 0 {
		exported := s.variables.Exported()
		sort.Strings(exported)
		for _, env := range exported {
			fmt.Printf("export %s\n", env)
		}
		return 0
	}

	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			name, value := parts[0], parts[1]
			s.variables.Set(name, value)
			s.variables.Export(name)
		} else {
			s.variables.Export(arg)
		}
	}

	return 0
}

func (s *Shell) builtinUnset(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "unset: not enough arguments\n")
		return 1
	}

	for _, arg := range args {
		if err := s.variables.Unset(arg); err != nil {
			fmt.Fprintf(os.Stderr, "unset: %v\n", err)
			return 1
		}
	}

	return 0
}

func (s *Shell) builtinSet(args []string) int {
	if len(args) == 0 {
		vars := s.variables.All()
		var names []string
		for name := range vars {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			v := vars[name]
			fmt.Printf("%s=%s\n", name, v.Value)
		}
		return 0
	}

	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			name, value := parts[0], parts[1]
			s.variables.Set(name, value)
		} else {
			switch arg {
			case "-e":
				s.config.POSIX = true
			case "+e":
				s.config.POSIX = false
			case "-x":
				s.config.Debug = true
			case "+x":
				s.config.Debug = false
			default:
				fmt.Printf("Unknown option: %s\n", arg)
				return 1
			}
		}
	}

	return 0
}

func (s *Shell) builtinSource(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "source: not enough arguments\n")
		return 1
	}

	filename := args[0]

	if !strings.Contains(filename, "/") {
		path := s.variables.Get("PATH")
		if path == "" {
			path = "/usr/local/bin:/usr/bin:/bin"
		}

		found := false
		for _, dir := range strings.Split(path, ":") {
			fullPath := filepath.Join(dir, filename)
			if _, err := os.Stat(fullPath); err == nil {
				filename = fullPath
				found = true
				break
			}
		}

		if !found {
			if _, err := os.Stat(filename); err != nil {
				fmt.Fprintf(os.Stderr, "source: %s: No such file or directory\n", filename)
				return 1
			}
		}
	}

	if err := s.sourceFile(filename); err != nil {
		fmt.Fprintf(os.Stderr, "source: %v\n", err)
		return 1
	}
	return 0
}

func (s *Shell) builtinJobs(args []string) int {
	s.jobs.Print()
	return 0
}

func (s *Shell) builtinFG(args []string) int {
	if len(args) == 0 {
		jobs := s.jobs.List()
		if len(jobs) == 0 {
			fmt.Fprintf(os.Stderr, "fg: no current job\n")
			return 1
		}

		for i := len(jobs) - 1; i >= 0; i-- {
			if jobs[i].State == s.jobs.Running()[0].State {
				if err := s.jobs.Foreground(jobs[i].ID); err != nil {
					fmt.Fprintf(os.Stderr, "fg: %v\n", err)
					return 1
				}
				return 0
			}
		}

		fmt.Fprintf(os.Stderr, "fg: no current job\n")
		return 1
	}

	jobID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "fg: %s: no such job\n", args[0])
		return 1
	}

	if err := s.jobs.Foreground(jobID); err != nil {
		fmt.Fprintf(os.Stderr, "fg: %v\n", err)
		return 1
	}

	return 0
}

func (s *Shell) builtinBG(args []string) int {
	if len(args) == 0 {
		jobs := s.jobs.Stopped()
		if len(jobs) == 0 {
			fmt.Fprintf(os.Stderr, "bg: no current job\n")
			return 1
		}

		if err := s.jobs.Background(jobs[len(jobs)-1].ID); err != nil {
			fmt.Fprintf(os.Stderr, "bg: %v\n", err)
			return 1
		}
		return 0
	}

	jobID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bg: %s: no such job\n", args[0])
		return 1
	}

	if err := s.jobs.Background(jobID); err != nil {
		fmt.Fprintf(os.Stderr, "bg: %v\n", err)
		return 1
	}

	return 0
}

func (s *Shell) builtinKill(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "kill: not enough arguments\n")
		return 1
	}

	for _, arg := range args {
		jobID, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: %s: no such job\n", arg)
			continue
		}

		if err := s.jobs.Kill(jobID); err != nil {
			fmt.Fprintf(os.Stderr, "kill: %v\n", err)
		}
	}

	return 0
}

func (s *Shell) builtinTest(args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "[: too few arguments\n")
		return 1
	}
	if args[len(args)-1] != "]" {
		fmt.Fprintf(os.Stderr, "[: missing ']'\n")
		return 1
	}
	left := args[0]
	op := args[1]
	right := args[2]
	switch op {
	case "-lt":
		l, _ := strconv.Atoi(left)
		r, _ := strconv.Atoi(right)
		if l < r {
			return 0
		}
		return 1
	case "-eq":
		l, _ := strconv.Atoi(left)
		r, _ := strconv.Atoi(right)
		if l == r {
			return 0
		}
		return 1
	default:
		fmt.Fprintf(os.Stderr, "[: unsupported op %s\n", op)
		return 1
	}
}
