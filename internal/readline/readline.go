package readline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"gosh/internal/history"

	"golang.org/x/term"
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
	state, err := makeRaw(int(os.Stdin.Fd()))
	if err != nil {
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
	defer restore(int(os.Stdin.Fd()), state)

	m.WriteString(prompt)

	var buf []rune
	histIdx := m.history.Size()
	pending := make([]byte, 0, 4)

	show := func() {
		m.WriteString("\r\033[K")
		m.WriteString(prompt)
		m.WriteString(string(buf))
	}

	for {
		var b [1]byte
		_, err := os.Stdin.Read(b[:])
		if err != nil {
			return "", err
		}
		byteVal := b[0]

		if len(pending) == 0 && (byteVal < 32 || byteVal == 127) {
			switch byteVal {
			case '\r', '\n':
				m.WriteString("\r\n")
				line := string(buf)
				if line != "" {
					m.history.Add(line)
				}
				return line, nil
			case 127, 8:
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
					show()
				}
				continue
			case 27:
				var seq [2]byte
				if _, err := os.Stdin.Read(seq[:]); err == nil && seq[0] == '[' {
					switch seq[1] {
					case 'A':
						if histIdx > 0 {
							histIdx--
							buf = []rune(m.history.Get(histIdx))
							show()
						}
					case 'B':
						if histIdx < m.history.Size()-1 {
							histIdx++
							buf = []rune(m.history.Get(histIdx))
						} else {
							histIdx = m.history.Size()
							buf = nil
						}
						show()
					}
				}
				continue
			case 3:
				m.WriteString("^C\r\n")
				return "", fmt.Errorf("interrupt")
			case 4:
				if len(buf) == 0 {
					m.WriteString("\r\n")
					return "", io.EOF
				}
				continue
			}
			continue
		}
		pending = append(pending, byteVal)
		if r, size := utf8.DecodeRune(pending); r != utf8.RuneError {
			if size == len(pending) {
				buf = append(buf, r)
				m.WriteString(string(pending))
				pending = pending[:0]
			}
		} else if len(pending) >= 4 {
			pending = pending[:0]
		}
	}
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
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return oldState, nil
}

func restore(fd int, state interface{}) error {
	if st, ok := state.(*term.State); ok {
		return term.Restore(fd, st)
	}
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
