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

		if p.pos < len(p.tokens) && p.current().Type == TokenSemicolon {
			p.advance()
		}
	}

	return commands, nil
}

func (p *Parser) parseCommand() (*ast.Command, error) {
	return p.parsePipeline()
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
			l.skipWhitespace()
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
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
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
