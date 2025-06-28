package prompt

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gosh/internal/variables"
)

type Manager struct {
	variables *variables.Manager
}

func New(vars *variables.Manager) *Manager {
	return &Manager{
		variables: vars,
	}
}

func (m *Manager) Generate(exitCode int) string {
	ps1 := m.variables.Get("PS1")
	if ps1 == "" {
		ps1 = "\\u@\\h:\\w\\$ "
	}

	return m.expandPrompt(ps1, exitCode)
}

func (m *Manager) GeneratePS2() string {
	ps2 := m.variables.Get("PS2")
	if ps2 == "" {
		ps2 = "> "
	}

	return m.expandPrompt(ps2, 0)
}

func (m *Manager) expandPrompt(prompt string, exitCode int) string {
	result := prompt

	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()
	pwd, _ := os.Getwd()
	home := os.Getenv("HOME")

	if strings.HasPrefix(pwd, home) {
		pwd = "~" + pwd[len(home):]
	}

	replacements := map[string]string{
		"\\u": currentUser.Username,
		"\\h": hostname,
		"\\H": hostname,
		"\\w": pwd,
		"\\W": filepath.Base(pwd),
		"\\d": time.Now().Format("Mon Jan 02"),
		"\\t": time.Now().Format("15:04:05"),
		"\\T": time.Now().Format("15:04:05"),
		"\\@": time.Now().Format("03:04 PM"),
		"\\A": time.Now().Format("15:04"),
		"\\n": "\n",
		"\\r": "\r",
		"\\$": func() string {
			if currentUser.Uid == "0" {
				return "#"
			}
			return "$"
		}(),
		"\\#":  fmt.Sprintf("%d", m.getCommandNumber()),
		"\\!":  fmt.Sprintf("%d", m.getHistoryNumber()),
		"\\j":  fmt.Sprintf("%d", m.getJobsCount()),
		"\\l":  m.getTTY(),
		"\\s":  "gosh",
		"\\v":  "1.0.0",
		"\\V":  "1.0.0",
		"\\\\": "\\",
	}

	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	if strings.Contains(result, "\\?") {
		result = strings.ReplaceAll(result, "\\?", fmt.Sprintf("%d", exitCode))
	}

	result = m.expandColors(result)

	return result
}

func (m *Manager) expandColors(prompt string) string {
	colorMap := map[string]string{
		"\\[\\033[0m\\]":  "\033[0m",  // reset
		"\\[\\033[1m\\]":  "\033[1m",  // bold
		"\\[\\033[30m\\]": "\033[30m", // black
		"\\[\\033[31m\\]": "\033[31m", // red
		"\\[\\033[32m\\]": "\033[32m", // green
		"\\[\\033[33m\\]": "\033[33m", // yellow
		"\\[\\033[34m\\]": "\033[34m", // blue
		"\\[\\033[35m\\]": "\033[35m", // magenta
		"\\[\\033[36m\\]": "\033[36m", // cyan
		"\\[\\033[37m\\]": "\033[37m", // white
		"\\[\\033[90m\\]": "\033[90m", // bright black
		"\\[\\033[91m\\]": "\033[91m", // bright red
		"\\[\\033[92m\\]": "\033[92m", // bright green
		"\\[\\033[93m\\]": "\033[93m", // bright yellow
		"\\[\\033[94m\\]": "\033[94m", // bright blue
		"\\[\\033[95m\\]": "\033[95m", // bright magenta
		"\\[\\033[96m\\]": "\033[96m", // bright cyan
		"\\[\\033[97m\\]": "\033[97m", // bright white
	}

	result := prompt
	for old, new := range colorMap {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

func (m *Manager) getCommandNumber() int {
	if cmd := m.variables.Get("BASH_COMMAND_NUMBER"); cmd != "" {
		var num int
		fmt.Sscanf(cmd, "%d", &num)
		return num
	}
	return 1
}

func (m *Manager) getHistoryNumber() int {
	if hist := m.variables.Get("HISTCMD"); hist != "" {
		var num int
		fmt.Sscanf(hist, "%d", &num)
		return num
	}
	return 1
}

func (m *Manager) getJobsCount() int {
	return 0
}

func (m *Manager) getTTY() string {
	return "console"
}

func (m *Manager) SetPS1(ps1 string) {
	m.variables.Set("PS1", ps1)
}

func (m *Manager) SetPS2(ps2 string) {
	m.variables.Set("PS2", ps2)
}

func (m *Manager) GetPS1() string {
	return m.variables.Get("PS1")
}

func (m *Manager) GetPS2() string {
	return m.variables.Get("PS2")
}
