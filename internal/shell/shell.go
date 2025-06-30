package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gosh/internal/builtin"
	"gosh/internal/config"
	"gosh/internal/executor"
	"gosh/internal/history"
	"gosh/internal/jobs"
	"gosh/internal/parser"
	"gosh/internal/prompt"
	"gosh/internal/readline"
	"gosh/internal/variables"
)

type Shell struct {
	config    *config.Config
	variables *variables.Manager
	executor  *executor.Executor
	parser    *parser.Parser
	history   *history.Manager
	prompt    *prompt.Manager
	readline  *readline.Manager
	builtins  *builtin.Manager
	jobs      *jobs.Manager

	interactive bool
	loginShell  bool
	exitCode    int
	running     bool

	currentDir string
	startTime  time.Time

	sigChan chan os.Signal
}

func New() *Shell {
	config := config.New()
	vars := variables.New()

	shell := &Shell{
		config:    config,
		variables: vars,
		parser:    parser.New(),
		history:   history.New(),
		prompt:    prompt.New(vars),
		builtins:  builtin.New(),
		jobs:      jobs.New(),

		interactive: false,
		loginShell:  false,
		exitCode:    0,
		running:     true,
		startTime:   time.Now(),
		sigChan:     make(chan os.Signal, 1),
	}

	shell.executor = executor.New(shell.variables, shell.builtins, shell.jobs)
	shell.readline = readline.New(shell.history)

	shell.initializeBuiltins()
	registerEaster(shell.builtins)
	shell.setupSignalHandlers()

	return shell
}

func (s *Shell) Run(args []string) error {
	if err := s.initialize(args); err != nil {
		return err
	}

	defer s.cleanup()

	if s.config.Command != "" {
		return s.executeCommand(s.config.Command)
	}

	if s.config.ScriptFile != "" {
		return s.executeScript(s.config.ScriptFile)
	}

	if s.config.ReadStdin {
		return s.readFromStdin()
	}

	if s.interactive {
		return s.interactiveLoop()
	}

	return s.readFromStdin()
}

func (s *Shell) initialize(args []string) error {
	if err := s.parseArguments(args); err != nil {
		return err
	}

	if err := s.initializeEnvironment(); err != nil {
		return err
	}

	if s.interactive && !s.config.NoRC {
		s.loadStartupFiles()
	}

	// env override: skip rc/profile if GOSH_NORC set
	if os.Getenv("GOSH_NORC") != "" {
		s.config.NoRC = true
		s.config.NoProfile = true
	}

	return nil
}

func (s *Shell) parseArguments(args []string) error {
	i := 1
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "-c":
			if i+1 >= len(args) {
				return fmt.Errorf("option -c requires an argument")
			}
			s.config.Command = args[i+1]
			i += 2
		case arg == "-i":
			s.interactive = true
			i++
		case arg == "-l" || arg == "--login":
			s.loginShell = true
			i++
		case arg == "-s":
			s.config.ReadStdin = true
			i++
		case arg == "--norc":
			s.config.NoRC = true
			i++
		case arg == "--noprofile":
			s.config.NoProfile = true
			i++
		case arg == "--posix":
			s.config.POSIX = true
			i++
		case arg == "--debug":
			s.config.Debug = true
			i++
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown option: %s", arg)
		default:
			s.config.ScriptFile = arg
			s.config.ScriptArgs = args[i:]
			break
		}
	}

	if s.config.Command == "" && s.config.ScriptFile == "" && !s.config.ReadStdin {
		s.interactive = true
	}

	return nil
}

func (s *Shell) initializeEnvironment() error {
	s.currentDir, _ = os.Getwd()

	s.variables.Set("PWD", s.currentDir)
	s.variables.Set("SHLVL", fmt.Sprintf("%d", s.getSHLVL()+1))
	s.variables.Set("GOSH_VERSION", "1.0.4")
	if execPath, err := os.Executable(); err == nil {
		s.variables.Set("SHELL", execPath)
	} else {
		s.variables.Set("SHELL", "gosh")
	}
	s.variables.Set("_", os.Args[0])

	if hostname, err := os.Hostname(); err == nil {
		s.variables.Set("HOSTNAME", hostname)
	}

	if user := os.Getenv("USER"); user != "" {
		s.variables.Set("USER", user)
	}

	if home := os.Getenv("HOME"); home != "" {
		s.variables.Set("HOME", home)
	}

	return nil
}

func (s *Shell) getSHLVL() int {
	if shlvl := os.Getenv("SHLVL"); shlvl != "" {
		if level := parseInt(shlvl); level > 0 {
			return level
		}
	}
	return 0
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func (s *Shell) loadStartupFiles() {
	if s.loginShell && !s.config.NoProfile {
		s.loadProfileFiles()
	}

	if !s.config.NoRC {
		s.loadRCFile()
	}
}

func (s *Shell) loadProfileFiles() {
	profiles := []string{
		"/etc/profile",
		filepath.Join(os.Getenv("HOME"), ".profile"),
		filepath.Join(os.Getenv("HOME"), ".bash_profile"),
		filepath.Join(os.Getenv("HOME"), ".gosh_profile"),
	}

	for _, profile := range profiles {
		if _, err := os.Stat(profile); err == nil {
			s.sourceFile(profile)
			break
		}
	}
}

func (s *Shell) loadRCFile() {
	rcFiles := []string{
		filepath.Join(os.Getenv("HOME"), ".goshrc"),
		filepath.Join(os.Getenv("HOME"), ".bashrc"),
	}

	for _, rcFile := range rcFiles {
		if _, err := os.Stat(rcFile); err == nil {
			s.sourceFile(rcFile)
			break
		}
	}
}

func (s *Shell) sourceFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		s.executeLine(line)
	}

	return scanner.Err()
}

func (s *Shell) setupSignalHandlers() {
	signal.Notify(s.sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	go func() {
		for sig := range s.sigChan {
			switch sig {
			case syscall.SIGINT:
				if s.interactive {
					fmt.Println()
					s.readline.ResetLine()
				} else {
					s.Exit(130)
				}
			case syscall.SIGTERM:
				s.Exit(143)
			case syscall.SIGTSTP:
				if s.interactive {
					s.suspendShell()
				}
			}
		}
	}()
}

func (s *Shell) interactiveLoop() error {
	fmt.Printf("gosh %s - Go Shell\n", s.variables.Get("GOSH_VERSION"))
	fmt.Println("Type 'help' for more information.")

	for s.running {
		promptStr := s.prompt.Generate(s.exitCode)

		line, err := s.readline.ReadLine(promptStr)
		if err != nil {
			if err == io.EOF {
				fmt.Println("exit")
				break
			}
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		s.history.Add(line)
		s.executeLine(line)
	}

	return nil
}

func (s *Shell) executeLine(line string) {
	commands, err := s.parser.Parse(line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
		s.exitCode = 2
		return
	}

	for _, cmd := range commands {
		exitCode := s.executor.Execute(cmd)
		s.exitCode = exitCode

		if s.config.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Command exit code: %d\n", exitCode)
		}
	}
}

func (s *Shell) executeCommand(command string) error {
	s.executeLine(command)
	s.Exit(s.exitCode)
	return nil
}

func (s *Shell) executeScript(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		s.executeLine(line)

		if !s.running {
			break
		}
	}

	s.Exit(s.exitCode)
	return scanner.Err()
}

func (s *Shell) readFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		s.executeLine(line)

		if !s.running {
			break
		}
	}

	return scanner.Err()
}

func (s *Shell) suspendShell() {
	fmt.Println("\n[Suspended]")
	syscall.Kill(os.Getpid(), syscall.SIGSTOP)
}

func (s *Shell) initializeBuiltins() {
	s.builtins.Register("exit", s.builtinExit)
	s.builtins.Register("cd", s.builtinCD)
	s.builtins.Register("pwd", s.builtinPWD)
	s.builtins.Register("echo", s.builtinEcho)
	s.builtins.Register("help", s.builtinHelp)
	s.builtins.Register("history", s.builtinHistory)
	s.builtins.Register("export", s.builtinExport)
	s.builtins.Register("unset", s.builtinUnset)
	s.builtins.Register("set", s.builtinSet)
	s.builtins.Register("source", s.builtinSource)
	s.builtins.Register(".", s.builtinSource)
	s.builtins.Register("jobs", s.builtinJobs)
	s.builtins.Register("fg", s.builtinFG)
	s.builtins.Register("bg", s.builtinBG)
	s.builtins.Register("kill", s.builtinKill)
	s.builtins.Register("[", s.builtinTest)
}

func (s *Shell) Exit(code int) {
	s.running = false
	s.cleanup()
	os.Exit(code)
}

func (s *Shell) cleanup() {
	if s.history != nil {
		s.history.Save()
	}
	if s.readline != nil {
		s.readline.Close()
	}
	close(s.sigChan)
}
