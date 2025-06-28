package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Manager struct {
	entries  []string
	file     string
	maxSize  int
	position int
}

func New() *Manager {
	home, _ := os.UserHomeDir()
	histFile := filepath.Join(home, ".gosh_history")

	m := &Manager{
		file:    histFile,
		maxSize: 1000,
	}

	m.Load()
	return m
}

func (m *Manager) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	if len(m.entries) > 0 && m.entries[len(m.entries)-1] == command {
		return
	}

	m.entries = append(m.entries, command)

	if len(m.entries) > m.maxSize {
		m.entries = m.entries[len(m.entries)-m.maxSize:]
	}

	m.position = len(m.entries)
}

func (m *Manager) Get(index int) string {
	if index >= 0 && index < len(m.entries) {
		return m.entries[index]
	}
	return ""
}

func (m *Manager) Previous() string {
	if m.position > 0 {
		m.position--
		return m.entries[m.position]
	}
	return ""
}

func (m *Manager) Next() string {
	if m.position < len(m.entries)-1 {
		m.position++
		return m.entries[m.position]
	} else if m.position == len(m.entries)-1 {
		m.position++
		return ""
	}
	return ""
}

func (m *Manager) Reset() {
	m.position = len(m.entries)
}

func (m *Manager) Search(query string) []string {
	var results []string
	for _, entry := range m.entries {
		if strings.Contains(entry, query) {
			results = append(results, entry)
		}
	}
	return results
}

func (m *Manager) All() []string {
	return append([]string{}, m.entries...)
}

func (m *Manager) Clear() {
	m.entries = nil
	m.position = 0
}

func (m *Manager) Size() int {
	return len(m.entries)
}

func (m *Manager) SetMaxSize(size int) {
	m.maxSize = size
	if len(m.entries) > size {
		m.entries = m.entries[len(m.entries)-size:]
		m.position = len(m.entries)
	}
}

func (m *Manager) Load() error {
	file, err := os.Open(m.file)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			m.entries = append(m.entries, line)
		}
	}

	if len(m.entries) > m.maxSize {
		m.entries = m.entries[len(m.entries)-m.maxSize:]
	}

	m.position = len(m.entries)
	return scanner.Err()
}

func (m *Manager) Save() error {
	file, err := os.Create(m.file)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, entry := range m.entries {
		if _, err := fmt.Fprintln(file, entry); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Expand(input string) (string, error) {
	if !strings.Contains(input, "!") {
		return input, nil
	}

	result := input

	if strings.Contains(result, "!!") {
		if len(m.entries) > 0 {
			last := m.entries[len(m.entries)-1]
			result = strings.ReplaceAll(result, "!!", last)
		} else {
			return "", fmt.Errorf("no previous command")
		}
	}

	if strings.Contains(result, "!") && len(result) > 1 {
		for i := 0; i < len(result)-1; i++ {
			if result[i] == '!' && result[i+1] >= '0' && result[i+1] <= '9' {
				end := i + 1
				for end < len(result) && result[end] >= '0' && result[end] <= '9' {
					end++
				}

				numStr := result[i+1 : end]
				if num, err := strconv.Atoi(numStr); err == nil {
					if num > 0 && num <= len(m.entries) {
						cmd := m.entries[num-1]
						result = result[:i] + cmd + result[end:]
						i += len(cmd) - 1
					}
				}
			}
		}
	}

	return result, nil
}

func (m *Manager) GetFile() string {
	return m.file
}

func (m *Manager) SetFile(file string) {
	m.file = file
}
