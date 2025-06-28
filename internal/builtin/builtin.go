package builtin

import (
	"fmt"
	"strconv"
	"strings"
)

type BuiltinFunc func(args []string) int

type Manager struct {
	builtins map[string]BuiltinFunc
}

func New() *Manager {
	return &Manager{
		builtins: make(map[string]BuiltinFunc),
	}
}

func (m *Manager) Register(name string, fn BuiltinFunc) {
	m.builtins[name] = fn
}

func (m *Manager) Get(name string) BuiltinFunc {
	return m.builtins[name]
}

func (m *Manager) List() []string {
	var names []string
	for name := range m.builtins {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Exists(name string) bool {
	_, exists := m.builtins[name]
	return exists
}

func (m *Manager) Remove(name string) {
	delete(m.builtins, name)
}

func ParseIntArg(arg string) (int, error) {
	return strconv.Atoi(arg)
}

func JoinArgs(args []string) string {
	return strings.Join(args, " ")
}

func PrintUsage(command string, usage string) {
	fmt.Printf("Usage: %s %s\n", command, usage)
}
