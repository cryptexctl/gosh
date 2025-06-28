package variables

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Variable struct {
	Name     string
	Value    string
	Exported bool
	ReadOnly bool
	Array    bool
	Values   []string
}

type Manager struct {
	vars map[string]*Variable
	mu   sync.RWMutex
}

func New() *Manager {
	m := &Manager{
		vars: make(map[string]*Variable),
	}

	m.loadEnvironment()
	return m
}

func (m *Manager) loadEnvironment() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			m.vars[parts[0]] = &Variable{
				Name:     parts[0],
				Value:    parts[1],
				Exported: true,
				ReadOnly: false,
			}
		}
	}
}

func (m *Manager) Set(name, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.vars[name]; exists && existing.ReadOnly {
		return fmt.Errorf("variable %s is read-only", name)
	}

	exported := false
	if existing, exists := m.vars[name]; exists {
		exported = existing.Exported
	}

	m.vars[name] = &Variable{
		Name:     name,
		Value:    value,
		Exported: exported,
		ReadOnly: false,
	}

	if exported {
		os.Setenv(name, value)
	}

	return nil
}

func (m *Manager) Get(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.vars[name]; exists {
		return v.Value
	}

	return os.Getenv(name)
}

func (m *Manager) Export(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, exists := m.vars[name]; exists {
		v.Exported = true
		os.Setenv(name, v.Value)
		return nil
	}

	value := os.Getenv(name)
	m.vars[name] = &Variable{
		Name:     name,
		Value:    value,
		Exported: true,
		ReadOnly: false,
	}

	return nil
}

func (m *Manager) Unset(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, exists := m.vars[name]; exists && v.ReadOnly {
		return fmt.Errorf("variable %s is read-only", name)
	}

	delete(m.vars, name)
	os.Unsetenv(name)

	return nil
}

func (m *Manager) SetReadOnly(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, exists := m.vars[name]; exists {
		v.ReadOnly = true
		return nil
	}

	return fmt.Errorf("variable %s not found", name)
}

func (m *Manager) IsExported(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.vars[name]; exists {
		return v.Exported
	}

	return false
}

func (m *Manager) IsReadOnly(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.vars[name]; exists {
		return v.ReadOnly
	}

	return false
}

func (m *Manager) All() map[string]*Variable {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Variable)
	for k, v := range m.vars {
		result[k] = &Variable{
			Name:     v.Name,
			Value:    v.Value,
			Exported: v.Exported,
			ReadOnly: v.ReadOnly,
			Array:    v.Array,
			Values:   append([]string{}, v.Values...),
		}
	}

	return result
}

func (m *Manager) Exported() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var exported []string
	for _, v := range m.vars {
		if v.Exported {
			exported = append(exported, fmt.Sprintf("%s=%s", v.Name, v.Value))
		}
	}

	sort.Strings(exported)
	return exported
}

func (m *Manager) SetArray(name string, values []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.vars[name]; exists && existing.ReadOnly {
		return fmt.Errorf("variable %s is read-only", name)
	}

	exported := false
	if existing, exists := m.vars[name]; exists {
		exported = existing.Exported
	}

	m.vars[name] = &Variable{
		Name:     name,
		Value:    strings.Join(values, " "),
		Exported: exported,
		ReadOnly: false,
		Array:    true,
		Values:   append([]string{}, values...),
	}

	if exported {
		os.Setenv(name, strings.Join(values, " "))
	}

	return nil
}

func (m *Manager) GetArray(name string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.vars[name]; exists && v.Array {
		return append([]string{}, v.Values...)
	}

	return nil
}

func (m *Manager) SetArrayElement(name string, index int, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.vars[name]; exists && existing.ReadOnly {
		return fmt.Errorf("variable %s is read-only", name)
	}

	v, exists := m.vars[name]
	if !exists {
		v = &Variable{
			Name:   name,
			Array:  true,
			Values: []string{},
		}
		m.vars[name] = v
	}

	if !v.Array {
		return fmt.Errorf("variable %s is not an array", name)
	}

	for len(v.Values) <= index {
		v.Values = append(v.Values, "")
	}

	v.Values[index] = value
	v.Value = strings.Join(v.Values, " ")

	if v.Exported {
		os.Setenv(name, v.Value)
	}

	return nil
}

func (m *Manager) GetArrayElement(name string, index int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.vars[name]; exists && v.Array {
		if index >= 0 && index < len(v.Values) {
			return v.Values[index]
		}
	}

	return ""
}

func (m *Manager) SubstituteVariables(text string) string {
	result := text

	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, variable := range m.vars {
		result = strings.ReplaceAll(result, "$"+name, variable.Value)
		result = strings.ReplaceAll(result, "${"+name+"}", variable.Value)
	}

	result = strings.ReplaceAll(result, "$$", strconv.Itoa(os.Getpid()))
	result = strings.ReplaceAll(result, "$?", "0")

	return result
}

func (m *Manager) EvalArithmetic(expr string) (int, error) {
	// very limited: supports VAR op INT or INT op VAR or INT op INT with + - * /
	expr = strings.TrimSpace(expr)
	ops := []string{"+", "-", "*", "/"}
	for _, op := range ops {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) != 2 {
				return 0, fmt.Errorf("bad expression")
			}
			aStr := strings.TrimSpace(parts[0])
			bStr := strings.TrimSpace(parts[1])
			aVal, err := m.arithOperand(aStr)
			if err != nil {
				return 0, err
			}
			bVal, err := m.arithOperand(bStr)
			if err != nil {
				return 0, err
			}
			switch op {
			case "+":
				return aVal + bVal, nil
			case "-":
				return aVal - bVal, nil
			case "*":
				return aVal * bVal, nil
			case "/":
				if bVal == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return aVal / bVal, nil
			}
		}
	}
	return m.arithOperand(expr)
}

func (m *Manager) arithOperand(tok string) (int, error) {
	if v, err := strconv.Atoi(tok); err == nil {
		return v, nil
	}
	val := m.Get(tok)
	return strconv.Atoi(val)
}
