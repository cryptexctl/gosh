package ast

type CommandType int

const (
	CommandSimple CommandType = iota
	CommandPipeline
	CommandBackground
	CommandList
	CommandIf
	CommandFor
	CommandWhile
	CommandCase
	CommandFunction
	CommandSubshell
	CommandGroup
)

type Command struct {
	Type       CommandType
	Simple     *SimpleCommand
	Pipeline   *Pipeline
	Background *BackgroundCommand
	List       *List
	If         *IfCommand
	For        *ForCommand
	While      *WhileCommand
	Case       *CaseCommand
	Function   *FunctionCommand
	Subshell   *SubshellCommand
	Group      *GroupCommand
}

type SimpleCommand struct {
	Name      string
	Args      []string
	Redirects []*Redirect
	Env       map[string]string
}

type Pipeline struct {
	Left  *Command
	Right *Command
}

type BackgroundCommand struct {
	Command *Command
}

type List struct {
	Commands  []*Command
	Operators []string
}

type IfCommand struct {
	Condition *Command
	Then      *Command
	Else      *Command
}

type ForCommand struct {
	Variable string
	Values   []string
	Body     *Command
}

type WhileCommand struct {
	Condition *Command
	Body      *Command
}

type CaseCommand struct {
	Word  string
	Cases []*CaseItem
}

type CaseItem struct {
	Patterns []string
	Command  *Command
}

type FunctionCommand struct {
	Name string
	Body *Command
}

type SubshellCommand struct {
	Command *Command
}

type GroupCommand struct {
	Commands []*Command
}

type RedirectType int

const (
	RedirectInput RedirectType = iota
	RedirectOutput
	RedirectAppend
	RedirectError
	RedirectErrorAppend
	RedirectInputOutput
	RedirectHereDoc
	RedirectHereString
)

type Redirect struct {
	Type    RedirectType
	Source  int
	Target  string
	HereDoc string
}

type Word struct {
	Text   string
	Quoted bool
}

type Expansion struct {
	Type  ExpansionType
	Value string
}

type ExpansionType int

const (
	ExpansionVariable ExpansionType = iota
	ExpansionCommand
	ExpansionArithmetic
	ExpansionGlob
)
