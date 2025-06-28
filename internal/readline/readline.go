package readline

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gosh/internal/history"
)

type Manager struct {
	history *history.Manager
	scanner *bufio.Scanner
	rawMode bool
}

func New(hist *history.Manager) *Manager {
	return &Manager{
		history: hist,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (m *Manager) ReadLine(prompt string) (string, error) {
	fmt.Print(prompt)

	if !m.scanner.Scan() {
		if err := m.scanner.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("EOF")
	}

	line := m.scanner.Text()
	return line, nil
}

func (m *Manager) ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	oldState, err := makeRaw(0)
	if err != nil {
		return "", err
	}
	defer restore(0, oldState)

	var password []byte
	var b [1]byte

	for {
		n, err := os.Stdin.Read(b[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		if b[0] == '\n' || b[0] == '\r' {
			break
		}

		if b[0] == 127 || b[0] == 8 {
			if len(password) > 0 {
				password = password[:len(password)-1]
			}
		} else {
			password = append(password, b[0])
		}
	}

	fmt.Println()
	return string(password), nil
}

func (m *Manager) ResetLine() {
	fmt.Print("\r\033[K")
}

func (m *Manager) Close() {
}

func (m *Manager) SetPrompt(prompt string) {
}

func (m *Manager) AddHistory(line string) {
	if m.history != nil {
		m.history.Add(line)
	}
}

func (m *Manager) LoadHistory() error {
	if m.history != nil {
		return m.history.Load()
	}
	return nil
}

func (m *Manager) SaveHistory() error {
	if m.history != nil {
		return m.history.Save()
	}
	return nil
}

func (m *Manager) ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

func (m *Manager) GetTerminalSize() (int, int) {
	return 80, 24
}

func (m *Manager) EnableRawMode() error {
	if m.rawMode {
		return nil
	}

	state, err := makeRaw(0)
	if err != nil {
		return err
	}

	m.rawMode = true
	_ = state
	return nil
}

func (m *Manager) DisableRawMode() error {
	if !m.rawMode {
		return nil
	}

	m.rawMode = false
	return nil
}

func (m *Manager) ReadChar() (rune, error) {
	var b [1]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, fmt.Errorf("no input")
	}
	return rune(b[0]), nil
}

func (m *Manager) WriteString(s string) {
	fmt.Print(s)
}

func (m *Manager) Refresh() {
}

func makeRaw(fd int) (interface{}, error) {
	return nil, fmt.Errorf("raw mode not supported on this platform")
}

func restore(fd int, state interface{}) error {
	return nil
}

func (m *Manager) SetCompletionCallback(callback func(string) []string) {
}

func (m *Manager) Complete(line string) []string {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	lastPart := parts[len(parts)-1]

	var completions []string

	if len(parts) == 1 {
		completions = append(completions, m.completeCommands(lastPart)...)
	} else {
		completions = append(completions, m.completeFiles(lastPart)...)
	}

	return completions
}

func (m *Manager) completeCommands(prefix string) []string {
	commands := []string{
		"cd", "pwd", "ls", "echo", "cat", "grep", "find", "which", "history",
		"exit", "help", "export", "unset", "set", "source", ".", "alias",
		"unalias", "jobs", "fg", "bg", "kill", "ps", "top", "date", "whoami",
	}

	var matches []string
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

func (m *Manager) completeFiles(prefix string) []string {
	dir := "."
	filename := prefix

	if strings.Contains(prefix, "/") {
		parts := strings.Split(prefix, "/")
		filename = parts[len(parts)-1]
		dir = strings.Join(parts[:len(parts)-1], "/")
		if dir == "" {
			dir = "/"
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), filename) {
			fullPath := entry.Name()
			if dir != "." {
				fullPath = dir + "/" + entry.Name()
			}
			if entry.IsDir() {
				fullPath += "/"
			}
			matches = append(matches, fullPath)
		}
	}

	return matches
}
