package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"gosh/internal/ast"
	"gosh/internal/builtin"
	"gosh/internal/jobs"
	"gosh/internal/variables"
)

type Executor struct {
	variables *variables.Manager
	builtins  *builtin.Manager
	jobs      *jobs.Manager

	lastExitCode int
}

func New(vars *variables.Manager, builtins *builtin.Manager, jobs *jobs.Manager) *Executor {
	return &Executor{
		variables:    vars,
		builtins:     builtins,
		jobs:         jobs,
		lastExitCode: 0,
	}
}

func (e *Executor) Execute(cmd *ast.Command) int {
	if cmd == nil {
		return 0
	}

	switch cmd.Type {
	case ast.CommandSimple:
		return e.executeSimple(cmd.Simple)
	case ast.CommandPipeline:
		return e.executePipeline(cmd.Pipeline)
	case ast.CommandBackground:
		return e.executeBackground(cmd.Background)
	case ast.CommandList:
		return e.executeList(cmd.List)
	case ast.CommandIf:
		return e.executeIf(cmd.If)
	case ast.CommandFor:
		return e.executeFor(cmd.For)
	case ast.CommandWhile:
		return e.executeWhile(cmd.While)
	case ast.CommandCase:
		return e.executeCase(cmd.Case)
	case ast.CommandFunction:
		return e.executeFunction(cmd.Function)
	case ast.CommandSubshell:
		return e.executeSubshell(cmd.Subshell)
	case ast.CommandGroup:
		return e.executeGroup(cmd.Group)
	default:
		return 1
	}
}

func (e *Executor) executeSimple(cmd *ast.SimpleCommand) int {
	if cmd == nil || cmd.Name == "" {
		return 0
	}

	name := e.variables.SubstituteVariables(cmd.Name)

	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		args[i] = e.variables.SubstituteVariables(arg)
	}

	if builtin := e.builtins.Get(name); builtin != nil {
		return builtin(args)
	}

	return e.executeExternal(name, args, cmd.Redirects)
}

func (e *Executor) executeExternal(name string, args []string, redirects []*ast.Redirect) int {
	cmdPath, err := e.findCommand(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %s: command not found\n", name)
		return 127
	}

	cmd := exec.Command(cmdPath, args...)

	cmd.Env = e.variables.Exported()

	if err := e.setupRedirects(cmd, redirects); err != nil {
		fmt.Fprintf(os.Stderr, "gosh: %v\n", err)
		return 1
	}

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}

	return 0
}

func (e *Executor) findCommand(name string) (string, error) {
	if strings.Contains(name, "/") {
		if _, err := os.Stat(name); err == nil {
			return name, nil
		}
		return "", fmt.Errorf("no such file or directory")
	}

	path := e.variables.Get("PATH")
	if path == "" {
		path = "/usr/local/bin:/usr/bin:/bin"
	}

	for _, dir := range strings.Split(path, ":") {
		cmdPath := filepath.Join(dir, name)
		if _, err := os.Stat(cmdPath); err == nil {
			return cmdPath, nil
		}
	}

	return "", fmt.Errorf("command not found")
}

func (e *Executor) setupRedirects(cmd *exec.Cmd, redirects []*ast.Redirect) error {
	for _, redirect := range redirects {
		switch redirect.Type {
		case ast.RedirectInput:
			file, err := os.Open(redirect.Target)
			if err != nil {
				return fmt.Errorf("cannot open %s: %v", redirect.Target, err)
			}
			cmd.Stdin = file

		case ast.RedirectOutput:
			file, err := os.Create(redirect.Target)
			if err != nil {
				return fmt.Errorf("cannot create %s: %v", redirect.Target, err)
			}
			cmd.Stdout = file

		case ast.RedirectAppend:
			file, err := os.OpenFile(redirect.Target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("cannot open %s: %v", redirect.Target, err)
			}
			cmd.Stdout = file

		case ast.RedirectError:
			file, err := os.Create(redirect.Target)
			if err != nil {
				return fmt.Errorf("cannot create %s: %v", redirect.Target, err)
			}
			cmd.Stderr = file
		}
	}

	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	return nil
}

func (e *Executor) executePipeline(pipeline *ast.Pipeline) int {
	if pipeline == nil {
		return 1
	}

	leftReader, leftWriter, err := os.Pipe()
	if err != nil {
		return 1
	}
	defer leftReader.Close()
	defer leftWriter.Close()

	var rightExitCode int

	done := make(chan bool, 2)

	go func() {
		defer leftWriter.Close()

		if pipeline.Left.Type == ast.CommandSimple {
			cmd := pipeline.Left.Simple
			if cmd != nil {
				cmdPath, err := e.findCommand(cmd.Name)
				if err == nil {
					execCmd := exec.Command(cmdPath, cmd.Args...)
					execCmd.Stdout = leftWriter
					execCmd.Stderr = os.Stderr
					execCmd.Stdin = os.Stdin
					execCmd.Env = e.variables.Exported()

					execCmd.Run()
				} else {
					// command lost
				}
			}
		} else {
			e.Execute(pipeline.Left)
		}
		done <- true
	}()

	go func() {
		defer leftReader.Close()

		if pipeline.Right.Type == ast.CommandSimple {
			cmd := pipeline.Right.Simple
			if cmd != nil {
				if builtin := e.builtins.Get(cmd.Name); builtin != nil {
					oldStdin := os.Stdin
					os.Stdin = leftReader
					rightExitCode = builtin(cmd.Args)
					os.Stdin = oldStdin
				} else {
					cmdPath, err := e.findCommand(cmd.Name)
					if err == nil {
						execCmd := exec.Command(cmdPath, cmd.Args...)
						execCmd.Stdin = leftReader
						execCmd.Stdout = os.Stdout
						execCmd.Stderr = os.Stderr
						execCmd.Env = e.variables.Exported()

						if err := execCmd.Run(); err != nil {
							if exitError, ok := err.(*exec.ExitError); ok {
								if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
									rightExitCode = status.ExitStatus()
								}
							} else {
								rightExitCode = 1
							}
						}
					} else {
						rightExitCode = 127
					}
				}
			}
		} else {
			oldStdin := os.Stdin
			os.Stdin = leftReader
			rightExitCode = e.Execute(pipeline.Right)
			os.Stdin = oldStdin
		}
		done <- true
	}()

	<-done
	<-done

	return rightExitCode
}

func (e *Executor) executeBackground(bg *ast.BackgroundCommand) int {
	if bg == nil {
		return 1
	}

	go func() {
		e.Execute(bg.Command)
	}()

	return 0
}

func (e *Executor) executeList(list *ast.List) int {
	if list == nil {
		return 0
	}

	var exitCode int
	for i, cmd := range list.Commands {
		exitCode = e.Execute(cmd)

		if i < len(list.Operators) {
			switch list.Operators[i] {
			case "&&":
				if exitCode != 0 {
					return exitCode
				}
			case "||":
				if exitCode == 0 {
					return exitCode
				}
			}
		}
	}

	return exitCode
}

func (e *Executor) executeIf(ifCmd *ast.IfCommand) int {
	if ifCmd == nil {
		return 1
	}

	conditionResult := e.Execute(ifCmd.Condition)

	if conditionResult == 0 {
		return e.Execute(ifCmd.Then)
	} else if ifCmd.Else != nil {
		return e.Execute(ifCmd.Else)
	}

	return 0
}

func (e *Executor) executeFor(forCmd *ast.ForCommand) int {
	if forCmd == nil {
		return 1
	}

	var exitCode int
	for _, value := range forCmd.Values {
		e.variables.Set(forCmd.Variable, value)
		exitCode = e.Execute(forCmd.Body)
	}

	return exitCode
}

func (e *Executor) executeWhile(whileCmd *ast.WhileCommand) int {
	if whileCmd == nil {
		return 1
	}

	var exitCode int
	for {
		conditionResult := e.Execute(whileCmd.Condition)
		if conditionResult != 0 {
			break
		}
		exitCode = e.Execute(whileCmd.Body)
	}

	return exitCode
}

func (e *Executor) executeCase(caseCmd *ast.CaseCommand) int {
	if caseCmd == nil {
		return 1
	}

	word := e.variables.SubstituteVariables(caseCmd.Word)

	for _, caseItem := range caseCmd.Cases {
		for _, pattern := range caseItem.Patterns {
			if matched, _ := filepath.Match(pattern, word); matched {
				return e.Execute(caseItem.Command)
			}
		}
	}

	return 0
}

func (e *Executor) executeFunction(funcCmd *ast.FunctionCommand) int {
	if funcCmd == nil {
		return 1
	}

	return 0
}

func (e *Executor) executeSubshell(subCmd *ast.SubshellCommand) int {
	if subCmd == nil {
		return 1
	}

	return e.Execute(subCmd.Command)
}

func (e *Executor) executeGroup(groupCmd *ast.GroupCommand) int {
	if groupCmd == nil {
		return 0
	}

	var exitCode int
	for _, cmd := range groupCmd.Commands {
		exitCode = e.Execute(cmd)
	}

	return exitCode
}

func (e *Executor) GetLastExitCode() int {
	return e.lastExitCode
}

func (e *Executor) SetLastExitCode(code int) {
	e.lastExitCode = code
}

func PipeCommands(commands [][]string) error {
	if len(commands) < 2 {
		return fmt.Errorf("pipe requires at least 2 commands")
	}

	var cmds []*exec.Cmd
	var pipes []io.WriteCloser

	for i, command := range commands {
		cmd := exec.Command(command[0], command[1:]...)
		cmds = append(cmds, cmd)

		if i > 0 {
			cmd.Stdin, _ = pipes[i-1].(io.Reader)
		}

		if i < len(commands)-1 {
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			pipes = append(pipes, stdout.(io.WriteCloser))
		} else {
			cmd.Stdout = os.Stdout
		}

		cmd.Stderr = os.Stderr
	}

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return err
		}
	}

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return err
		}
	}

	return nil
}
