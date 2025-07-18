package parser

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"gosh/internal/ast"
)

type Parser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(input string) ([]*ast.Command, error) {
	p.lexer = NewLexer(input)
	p.tokens = p.lexer.Tokenize()
	p.pos = 0

	var commands []*ast.Command

	for p.pos < len(p.tokens) {
		if p.current().Type == TokenNewline {
			p.advance()
			continue
		}
		if p.current().Type == TokenEOF {
			break
		}

		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}

		if cmd != nil {
			commands = append(commands, cmd)
		}

		if p.pos < len(p.tokens) && (p.current().Type == TokenSemicolon || p.current().Type == TokenNewline) {
			p.advance()
		}
	}

	return commands, nil
}

func (p *Parser) parseCommand() (*ast.Command, error) {
	if tok := p.current(); tok.Type == TokenWord {
		switch tok.Value {
		case "if":
			return p.parseIf()
		case "while":
			return p.parseWhile()
		case "for":
			return p.parseFor()
		}
	}
	left, err := p.parsePipeline()
	if err != nil {
		return nil, err
	}
	cmds := []*ast.Command{left}
	ops := []string{}
	for p.pos < len(p.tokens) && (p.current().Type == TokenAnd || p.current().Type == TokenOr) {
		opTok := p.current()
		p.advance()
		right, err := p.parsePipeline()
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, right)
		if opTok.Type == TokenAnd {
			ops = append(ops, "&&")
		} else {
			ops = append(ops, "||")
		}
	}
	if len(cmds) == 1 {
		return left, nil
	}
	return &ast.Command{Type: ast.CommandList, List: &ast.List{Commands: cmds, Operators: ops}}, nil
}

func (p *Parser) parsePipeline() (*ast.Command, error) {
	left, err := p.parseSimpleCommand()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) && p.current().Type == TokenPipe {
		p.advance()

		right, err := p.parseSimpleCommand()
		if err != nil {
			return nil, err
		}

		left = &ast.Command{
			Type: ast.CommandPipeline,
			Pipeline: &ast.Pipeline{
				Left:  left,
				Right: right,
			},
		}
	}

	return left, nil
}

func (p *Parser) parseSimpleCommand() (*ast.Command, error) {
	var args []string
	var redirects []*ast.Redirect

	for p.pos < len(p.tokens) {
		token := p.current()

		switch token.Type {
		case TokenWord:
			args = append(args, token.Value)
			p.advance()
		case TokenRedirectOut, TokenRedirectIn, TokenRedirectAppend:
			redirect, err := p.parseRedirect()
			if err != nil {
				return nil, err
			}
			redirects = append(redirects, redirect)
		case TokenNewline:
			goto done
		default:
			goto done
		}
	}

done:
	if len(args) == 0 {
		return nil, nil
	}

	return &ast.Command{
		Type: ast.CommandSimple,
		Simple: &ast.SimpleCommand{
			Name:      args[0],
			Args:      args[1:],
			Redirects: redirects,
		},
	}, nil
}

func (p *Parser) parseRedirect() (*ast.Redirect, error) {
	token := p.current()
	p.advance()

	if p.pos >= len(p.tokens) || p.current().Type != TokenWord {
		return nil, fmt.Errorf("expected filename after redirect")
	}

	target := p.current().Value
	p.advance()

	var redirectType ast.RedirectType
	switch token.Type {
	case TokenRedirectOut:
		redirectType = ast.RedirectOutput
	case TokenRedirectIn:
		redirectType = ast.RedirectInput
	case TokenRedirectAppend:
		redirectType = ast.RedirectAppend
	}

	return &ast.Redirect{
		Type:   redirectType,
		Target: target,
	}, nil
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

type TokenType int

const (
	TokenWord TokenType = iota
	TokenPipe
	TokenRedirectOut
	TokenRedirectIn
	TokenRedirectAppend
	TokenSemicolon
	TokenNewline
	TokenAnd
	TokenOr
	TokenBackground
	TokenEOF
)

type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
	}
}

func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		if unicode.IsSpace(rune(l.input[l.pos])) {
			if l.input[l.pos] == '\n' {
				l.addToken(TokenNewline, "\n")
				l.pos++
			} else {
				l.skipWhitespace()
			}
			continue
		}

		switch l.input[l.pos] {
		case '|':
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '|' {
				l.addToken(TokenOr, "||")
				l.pos += 2
			} else {
				l.addToken(TokenPipe, "|")
				l.pos++
			}
		case '&':
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '&' {
				l.addToken(TokenAnd, "&&")
				l.pos += 2
			} else {
				l.addToken(TokenBackground, "&")
				l.pos++
			}
		case '>':
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '>' {
				l.addToken(TokenRedirectAppend, ">>")
				l.pos += 2
			} else {
				l.addToken(TokenRedirectOut, ">")
				l.pos++
			}
		case '<':
			l.addToken(TokenRedirectIn, "<")
			l.pos++
		case ';':
			l.addToken(TokenSemicolon, ";")
			l.pos++
		case '"', '\'':
			l.tokenizeQuotedString()
		case '#':
			l.skipComment()
		default:
			l.tokenizeWord()
		}
	}

	l.addToken(TokenEOF, "")
	return l.tokens
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) tokenizeWord() {
	start := l.pos

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsSpace(rune(ch)) || ch == '|' || ch == '&' || ch == '>' || ch == '<' || ch == ';' {
			break
		}
		l.pos++
	}

	word := l.input[start:l.pos]
	l.addToken(TokenWord, word)
}

func (l *Lexer) tokenizeQuotedString() {
	quote := l.input[l.pos]
	l.pos++
	start := l.pos

	for l.pos < len(l.input) && l.input[l.pos] != quote {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos += 2
		} else {
			l.pos++
		}
	}

	if l.pos >= len(l.input) {
		l.addToken(TokenWord, l.input[start:])
		return
	}

	word := l.input[start:l.pos]
	l.pos++
	l.addToken(TokenWord, word)
}

func (l *Lexer) addToken(tokenType TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:  tokenType,
		Value: value,
		Pos:   l.pos,
	})
}

func ExpandVariables(text string, getVar func(string) string) string {
	arithRe := regexp.MustCompile(`\$\(\(([^)]+)\)\)`)
	text = arithRe.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimSpace(m[3 : len(m)-2])
		if getVar == nil {
			return m
		}
		varRe := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
		replaced := varRe.ReplaceAllStringFunc(inner, func(v string) string {
			return getVar(v)
		})
		parts := strings.FieldsFunc(replaced, func(r rune) bool {
			return r == '+' || r == '-' || r == '*' || r == '/'
		})
		if len(parts) == 2 {
			inner = replaced
		}
		return replaced
	})

	varRegex := regexp.MustCompile(`\$(\w+)|\$\{([^}]+)\}`)
	return varRegex.ReplaceAllStringFunc(text, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}

		if value := getVar(varName); value != "" {
			return value
		}
		return match
	})
}

func ExpandGlobs(pattern string) ([]string, error) {
	if !strings.ContainsAny(pattern, "*?[]") {
		return []string{pattern}, nil
	}

	return []string{pattern}, nil
}

func (p *Parser) parseIf() (*ast.Command, error) {
	p.advance()

	condTokens := []Token{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "then") {
		condTokens = append(condTokens, p.current())
		p.advance()
	}
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'then' keyword in if statement")
	}
	p.advance()

	thenTokens := []Token{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && (p.current().Value == "else" || p.current().Value == "fi")) {
		thenTokens = append(thenTokens, p.current())
		p.advance()
	}

	var elseTokens []Token
	if p.pos < len(p.tokens) && p.current().Type == TokenWord && p.current().Value == "else" {
		p.advance() // skip 'else'
		for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "fi") {
			elseTokens = append(elseTokens, p.current())
			p.advance()
		}
	}

	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'fi' to close if statement")
	}
	p.advance() // skip 'fi'

	condParser := &Parser{tokens: condTokens, pos: 0}
	condCmds, err := condParser.parsePipeline()
	if err != nil {
		return nil, err
	}
	thenParser := &Parser{tokens: thenTokens, pos: 0}
	thenCmd, err := thenParser.Parse(strings.Join(tokensToStrings(thenTokens), " "))
	if err != nil {
		return nil, err
	}
	var thenCmdNode *ast.Command
	if len(thenCmd) > 0 {
		thenCmdNode = thenCmd[0]
	}

	var elseCmdNode *ast.Command
	if len(elseTokens) > 0 {
		elseParser := &Parser{tokens: elseTokens, pos: 0}
		elseCmds, _ := elseParser.Parse(strings.Join(tokensToStrings(elseTokens), " "))
		if len(elseCmds) > 0 {
			elseCmdNode = elseCmds[0]
		}
	}

	var elifTokens [][]Token
	for p.pos < len(p.tokens) && p.current().Type == TokenWord && p.current().Value == "elif" {
		p.advance() // skip 'elif'
		condElif := []Token{}
		for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "then") {
			condElif = append(condElif, p.current())
			p.advance()
		}
		if p.pos >= len(p.tokens) {
			return nil, fmt.Errorf("expected 'then' in elif")
		}
		p.advance() // skip 'then'
		bodyElif := []Token{}
		for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && (p.current().Value == "elif" || p.current().Value == "else" || p.current().Value == "fi")) {
			bodyElif = append(bodyElif, p.current())
			p.advance()
		}
		elifTokens = append(elifTokens, append(condElif, Token{Type: TokenSemicolon, Value: ";;"}))
		elifTokens = append(elifTokens, bodyElif)
	}

	return &ast.Command{
		Type: ast.CommandIf,
		If: &ast.IfCommand{
			Condition: condCmds,
			Then:      thenCmdNode,
			Else:      elseCmdNode,
		},
	}, nil
}

func tokensToStrings(ts []Token) []string {
	var s []string
	for _, t := range ts {
		s = append(s, t.Value)
	}
	return s
}

func (p *Parser) parseWhile() (*ast.Command, error) {
	p.advance()
	condTokens := []Token{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "do") {
		condTokens = append(condTokens, p.current())
		p.advance()
	}
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'do' in while")
	}
	p.advance()

	bodyTokens := []Token{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "done") {
		bodyTokens = append(bodyTokens, p.current())
		p.advance()
	}
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'done' in while")
	}
	p.advance()

	condParser := &Parser{tokens: condTokens, pos: 0}
	condCmd, _ := condParser.parsePipeline()
	bodyParser := &Parser{tokens: bodyTokens, pos: 0}
	bodyCmds, _ := bodyParser.Parse(strings.Join(tokensToStrings(bodyTokens), " "))
	var bodyCmd *ast.Command
	if len(bodyCmds) > 0 {
		bodyCmd = bodyCmds[0]
	}
	return &ast.Command{
		Type: ast.CommandWhile,
		While: &ast.WhileCommand{
			Condition: condCmd,
			Body:      bodyCmd,
		},
	}, nil
}

func (p *Parser) parseFor() (*ast.Command, error) {
	p.advance()
	if p.current().Type != TokenWord {
		return nil, fmt.Errorf("expected variable after for")
	}
	varName := p.current().Value
	p.advance()
	if !(p.current().Type == TokenWord && p.current().Value == "in") {
		return nil, fmt.Errorf("expected 'in' after for variable")
	}
	p.advance()

	values := []string{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "do") {
		if p.current().Type == TokenWord {
			values = append(values, p.current().Value)
		}
		p.advance()
	}
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'do' in for")
	}
	p.advance()

	bodyTokens := []Token{}
	for p.pos < len(p.tokens) && !(p.current().Type == TokenWord && p.current().Value == "done") {
		bodyTokens = append(bodyTokens, p.current())
		p.advance()
	}
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("expected 'done' to close for")
	}
	p.advance()

	bodyParser := &Parser{tokens: bodyTokens, pos: 0}
	bodyCmds, _ := bodyParser.Parse(strings.Join(tokensToStrings(bodyTokens), " "))
	var bodyCmd *ast.Command
	if len(bodyCmds) > 0 {
		bodyCmd = bodyCmds[0]
	}

	return &ast.Command{
		Type: ast.CommandFor,
		For: &ast.ForCommand{
			Variable: varName,
			Values:   values,
			Body:     bodyCmd,
		},
	}, nil
}
