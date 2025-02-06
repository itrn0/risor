// Package parser is used to generate the abstract syntax tree (AST) for a program.
//
// A parser is created by calling New() with a lexer as input. The parser should
// then be used only once, by calling parser.Parse() to produce the AST.
package parser

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/itrn0/risor/ast"
	"github.com/itrn0/risor/internal/tmpl"
	"github.com/itrn0/risor/lexer"
	"github.com/itrn0/risor/token"
)

type (
	prefixParseFn  func() ast.Node
	infixParseFn   func(ast.Node) ast.Node
	postfixParseFn func() ast.Statement
)

var statementTerminators = map[token.Type]bool{
	token.SEMICOLON:   true,
	token.NEWLINE:     true,
	token.RBRACE:      true,
	token.EOF:         true,
	token.PLUS_PLUS:   true,
	token.MINUS_MINUS: true,
}

// Parse the provided input as Risor source code and return the AST. This is
// shorthand way to create a Lexer and Parser and then call Parse on that.
func Parse(ctx context.Context, input string, options ...Option) (*ast.Program, error) {
	l := lexer.New(input)
	p := New(l, options...)
	if p.filename != "" {
		// If an option specified a filename, pass that through to the lexer.
		l.SetFilename(p.filename)
	}
	return p.Parse(ctx)
}

// Option is a configuration function for a Lexer.
type Option func(*Parser)

// WithFile sets the file name for the Lexer.
func WithFile(file string) Option {
	return func(l *Parser) {
		l.filename = file
	}
}

// Parser object
type Parser struct {
	// the Context supplied in the Parse() call
	ctx context.Context

	// l is our lexer
	l *lexer.Lexer

	// prevToken holds the previous token, which we already processed.
	prevToken token.Token

	// curToken holds the current token from the lexer.
	curToken token.Token

	// peekToken holds the next token from the lexer.
	peekToken token.Token

	// the parsing error, if any
	err ParserError

	// prefixParseFns holds a map of parsing methods for
	// prefix-based syntax.
	prefixParseFns map[token.Type]prefixParseFn

	// infixParseFns holds a map of parsing methods for
	// infix-based syntax.
	infixParseFns map[token.Type]infixParseFn

	// postfixParseFns holds a map of parsing methods for
	// postfix-based syntax.
	postfixParseFns map[token.Type]postfixParseFn

	// are we inside a ternary expression?
	//
	// Nested ternary expressions are illegal :)
	tern bool

	// The filename of the input
	filename string
}

// New returns a Parser for the program provided by the given Lexer.
func New(l *lexer.Lexer, options ...Option) *Parser {
	// Create the parser and apply any provided options
	p := &Parser{
		l:               l,
		prefixParseFns:  map[token.Type]prefixParseFn{},
		infixParseFns:   map[token.Type]infixParseFn{},
		postfixParseFns: map[token.Type]postfixParseFn{},
	}
	for _, opt := range options {
		opt(p)
	}

	// Prime the token pump
	p.nextToken() // makes curToken=<empty>, peekToken=token[0]
	p.nextToken() // makes curToken=token[0], peekToken=token[1]

	// Register prefix-functions
	p.registerPrefix(token.BACKTICK, p.parseString)
	p.registerPrefix(token.BANG, p.parsePrefixExpr)
	p.registerPrefix(token.DEFER, p.parseDefer)
	p.registerPrefix(token.EOF, p.illegalToken)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.FLOAT, p.parseFloat)
	p.registerPrefix(token.FOR, p.parseFor)
	p.registerPrefix(token.FROM, p.parseFromImport)
	p.registerPrefix(token.FSTRING, p.parseString)
	p.registerPrefix(token.FUNC, p.parseFunc)
	p.registerPrefix(token.GO, p.parseGo)
	p.registerPrefix(token.IDENT, p.parseIdent)
	p.registerPrefix(token.IF, p.parseIf)
	p.registerPrefix(token.ILLEGAL, p.illegalToken)
	p.registerPrefix(token.IMPORT, p.parseImport)
	p.registerPrefix(token.INT, p.parseInt)
	p.registerPrefix(token.LBRACE, p.parseMapOrSet)
	p.registerPrefix(token.LBRACKET, p.parseList)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpr)
	p.registerPrefix(token.MINUS, p.parsePrefixExpr)
	p.registerPrefix(token.NEWLINE, p.parseNewline)
	p.registerPrefix(token.NIL, p.parseNil)
	p.registerPrefix(token.PIPE, p.parsePrefixExpr)
	p.registerPrefix(token.RANGE, p.parseRange)
	p.registerPrefix(token.STRING, p.parseString)
	p.registerPrefix(token.SWITCH, p.parseSwitch)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.SEND, p.parseReceive)

	// Register infix functions
	p.registerInfix(token.AND, p.parseInfixExpr)
	p.registerInfix(token.ASSIGN, p.parseAssign)
	p.registerInfix(token.ASTERISK_EQUALS, p.parseAssign)
	p.registerInfix(token.ASTERISK, p.parseInfixExpr)
	p.registerInfix(token.EQ, p.parseInfixExpr)
	p.registerInfix(token.GT_EQUALS, p.parseInfixExpr)
	p.registerInfix(token.GT_GT, p.parseInfixExpr)
	p.registerInfix(token.GT, p.parseInfixExpr)
	p.registerInfix(token.IN, p.parseIn)
	p.registerInfix(token.LBRACKET, p.parseIndex)
	p.registerInfix(token.LPAREN, p.parseCall)
	p.registerInfix(token.LT_EQUALS, p.parseInfixExpr)
	p.registerInfix(token.LT_LT, p.parseInfixExpr)
	p.registerInfix(token.LT, p.parseInfixExpr)
	p.registerInfix(token.MINUS_EQUALS, p.parseAssign)
	p.registerInfix(token.MINUS, p.parseInfixExpr)
	p.registerInfix(token.MOD, p.parseInfixExpr)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpr)
	p.registerInfix(token.OR, p.parseInfixExpr)
	p.registerInfix(token.PERIOD, p.parseGetAttr)
	p.registerInfix(token.PIPE, p.parsePipe)
	p.registerInfix(token.PLUS_EQUALS, p.parseAssign)
	p.registerInfix(token.PLUS, p.parseInfixExpr)
	p.registerInfix(token.POW, p.parseInfixExpr)
	p.registerInfix(token.QUESTION, p.parseTernary)
	p.registerInfix(token.SEND, p.parseSend)
	p.registerInfix(token.SLASH_EQUALS, p.parseAssign)
	p.registerInfix(token.SLASH, p.parseInfixExpr)

	// Register postfix functions
	p.registerPostfix(token.MINUS_MINUS, p.parsePostfix)
	p.registerPostfix(token.PLUS_PLUS, p.parsePostfix)
	return p
}

// nextToken moves to the next token from the lexer, updating all of
// prevToken, curToken, and peekToken.
func (p *Parser) nextToken() error {
	// If we have an error, we can't move forward
	if p.err != nil {
		return p.err
	}
	var err error
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken, err = p.l.Next()
	if err == nil {
		return nil // success
	}
	// The lexer encountered an error. We consider all lexer errors
	// "syntax errors" and parsing will now be considered broken.
	p.err = NewSyntaxError(ErrorOpts{
		Cause:         err,
		File:          p.l.Filename(),
		StartPosition: p.peekToken.StartPosition,
		EndPosition:   p.peekToken.EndPosition,
		SourceCode:    p.l.GetLineText(p.peekToken),
	})
	return p.err
}

// Parse the program that is provided via the lexer.
func (p *Parser) Parse(ctx context.Context) (*ast.Program, error) {
	p.ctx = ctx
	// It's possible for an error to already exist because we read tokens from
	// the lexer in the constructor. Parsing is already broken if so.
	if p.err != nil {
		return nil, p.err
	}
	// Parse the entire input program as a series of statements.
	// Parsing stops on the first occurrence of an error.
	var statements []ast.Node
	for p.curToken.Type != token.EOF {
		// Check for context timeout
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		stmt := p.parseStatementStrict()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		if err := p.nextToken(); err != nil {
			return nil, err
		}
	}
	return ast.NewProgram(statements), p.err
}

// registerPrefix registers a function for handling a prefix-based statement.
func (p *Parser) registerPrefix(tokenType token.Type, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix registers a function for handling an infix-based statement.
func (p *Parser) registerInfix(tokenType token.Type, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// registerPostfix registers a function for handling a postfix-based statement.
func (p *Parser) registerPostfix(tokenType token.Type, fn postfixParseFn) {
	p.postfixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t token.Token) {
	if p.err != nil {
		return
	}
	p.err = NewParserError(ErrorOpts{
		ErrType:       "parse error",
		Message:       fmt.Sprintf("invalid syntax (unexpected %q)", t.Literal),
		File:          p.l.Filename(),
		StartPosition: t.StartPosition,
		EndPosition:   t.EndPosition,
		SourceCode:    p.l.GetLineText(t),
	})
}

// peekError raises an error if the next token is not the expected type.
func (p *Parser) peekError(context string, expected token.Type, got token.Token) {
	if p.err != nil {
		return
	}
	gotDesc := tokenDescription(got)
	expDesc := tokenTypeDescription(expected)
	p.err = NewParserError(ErrorOpts{
		ErrType: "parse error",
		Message: fmt.Sprintf("unexpected %s while parsing %s (expected %s)",
			gotDesc, context, expDesc),
		File:          p.l.Filename(),
		StartPosition: got.StartPosition,
		EndPosition:   got.EndPosition,
		SourceCode:    p.l.GetLineText(got),
	})
}

func (p *Parser) setError(err ParserError) {
	if p.err != nil {
		return
	}
	p.err = err
}

func (p *Parser) parseStatementStrict() ast.Node {
	stmt := p.parseStatement()
	if stmt == nil {
		return nil
	}
	// statement should end with a semicolon or the next token should be
	// a statement terminator
	if !p.curTokenIs(token.SEMICOLON) && !statementTerminators[p.peekToken.Type] {
		p.setTokenError(p.curToken, "unexpected token %q following statement", p.peekToken.Literal)
		return nil
	}
	return stmt
}

func (p *Parser) parseStatement() ast.Node {
	var stmt ast.Node
	switch p.curToken.Type {
	case token.VAR:
		stmt = p.parseVar()
	case token.CONST:
		stmt = p.parseConst()
	case token.RETURN:
		stmt = p.parseReturn()
	case token.BREAK:
		stmt = p.parseBreak()
	case token.CONTINUE:
		stmt = p.parseContinue()
	case token.NEWLINE:
		stmt = nil
	case token.IDENT:
		if p.peekTokenIs(token.DECLARE) || p.peekTokenIs(token.COMMA) {
			stmt = p.parseDeclaration()
		} else {
			stmt = p.parseExpressionStatement()
		}
	default:
		stmt = p.parseExpressionStatement()
	}
	// Consume trailing semicolon if present
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseVar() ast.Node {
	tok := p.curToken
	if !p.expectPeek("var statement", token.IDENT) {
		return nil
	}
	idents := []*ast.Ident{ast.NewIdent(p.curToken)}
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if !p.expectPeek("var statement", token.IDENT) {
			return nil
		}
		idents = append(idents, ast.NewIdent(p.curToken))
	}
	if !p.expectPeek("var statement", token.ASSIGN) {
		return nil
	}
	p.nextToken()
	value := p.parseAssignmentValue()
	if value == nil {
		return nil
	}
	if len(idents) > 1 {
		return ast.NewMultiVar(tok, idents, value, false)
	}
	return ast.NewVar(tok, idents[0], value)
}

func (p *Parser) parseDeclaration() ast.Node {
	tok := p.curToken
	idents := []*ast.Ident{ast.NewIdent(p.curToken)}
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if !p.expectPeek("declaration statement", token.IDENT) {
			return nil
		}
		idents = append(idents, ast.NewIdent(p.curToken))
	}
	var isWalrus bool
	switch p.peekToken.Type {
	case token.ASSIGN:
		isWalrus = false
	case token.DECLARE:
		isWalrus = true
	default:
		p.expectPeek("declaration statement", token.ASSIGN)
		return nil
	}
	p.nextToken() // move to the assignment operator
	p.nextToken() // move to the value
	value := p.parseAssignmentValue()
	if value == nil {
		return nil
	}
	if len(idents) > 1 {
		return ast.NewMultiVar(tok, idents, value, isWalrus)
	}
	return ast.NewDeclaration(tok, idents[0], value)
}

func (p *Parser) parseConst() *ast.Const {
	tok := p.curToken
	if !p.expectPeek("const statement", token.IDENT) {
		return nil
	}
	ident := ast.NewIdent(p.curToken)
	if !p.expectPeek("const statement", token.ASSIGN) {
		return nil
	}
	p.nextToken()
	value := p.parseAssignmentValue()
	if value == nil {
		return nil
	}
	return ast.NewConst(tok, ident, value)
}

// Parses the right hand side of an assignment statement.
func (p *Parser) parseAssignmentValue() ast.Expression {
	result := p.parseExpression(LOWEST)
	if result == nil {
		p.setError(NewParserError(ErrorOpts{
			ErrType:       "parse error",
			Message:       "assignment is missing a value",
			File:          p.l.Filename(),
			StartPosition: p.prevToken.EndPosition,
			EndPosition:   p.prevToken.EndPosition,
			SourceCode:    p.l.GetLineText(p.prevToken),
		}))
		return nil
	}
	return result
}

func (p *Parser) parseReturn() *ast.Return {
	returnToken := p.curToken
	if p.peekTokenIs(token.SEMICOLON) ||
		p.peekTokenIs(token.NEWLINE) ||
		p.peekTokenIs(token.RBRACE) ||
		p.peekTokenIs(token.EOF) {
		return ast.NewReturn(returnToken, nil)
	}
	p.nextToken()
	value := p.parseExpression(LOWEST)
	if value == nil {
		return nil
	}
	return ast.NewReturn(returnToken, value)
}

func (p *Parser) parseBreak() *ast.Control {
	return ast.NewControl(p.curToken, nil)
}

func (p *Parser) parseContinue() *ast.Control {
	return ast.NewControl(p.curToken, nil)
}

func (p *Parser) parseExpressionStatement() ast.Node {
	expr := p.parseNode(LOWEST)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid syntax")
		return nil
	}
	return expr
}

func (p *Parser) parseNode(precedence int) ast.Node {
	if p.curToken.Type == token.EOF || p.err != nil {
		return nil
	}
	postfix := p.postfixParseFns[p.curToken.Type]
	if postfix != nil {
		return postfix()
	}
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken)
		return nil
	}
	leftExp := prefix()
	if p.err != nil || leftExp == nil {
		return nil
	}
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		if err := p.nextToken(); err != nil {
			return nil
		}
		leftExp = infix(leftExp)
		if p.err != nil {
			break
		}
	}
	return leftExp
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	node := p.parseNode(precedence)
	if node == nil {
		return nil
	}
	if p.err != nil {
		return nil
	}
	if expr, ok := node.(ast.Expression); ok {
		return expr
	}
	p.setTokenError(p.prevToken, "expected expression")
	return nil
}

func (p *Parser) illegalToken() ast.Node {
	p.setError(NewParserError(ErrorOpts{
		ErrType:       "parse error",
		Message:       fmt.Sprintf("illegal token %s", p.curToken.Literal),
		File:          p.l.Filename(),
		StartPosition: p.curToken.StartPosition,
		EndPosition:   p.curToken.EndPosition,
		SourceCode:    p.l.GetLineText(p.curToken),
	}))
	return nil
}

func (p *Parser) setTokenError(t token.Token, msg string, args ...interface{}) ast.Node {
	p.setError(NewParserError(ErrorOpts{
		ErrType:       "parse error",
		Message:       fmt.Sprintf(msg, args...),
		File:          p.l.Filename(),
		StartPosition: t.StartPosition,
		EndPosition:   t.EndPosition,
		SourceCode:    p.l.GetLineText(t),
	}))
	return nil
}

func (p *Parser) parseIdent() ast.Node {
	if p.curToken.Literal == "" {
		p.setTokenError(p.curToken, "invalid identifier")
		return nil
	}
	return ast.NewIdent(p.curToken)
}

func (p *Parser) parseInt() ast.Node {
	tok, lit := p.curToken, p.curToken.Literal
	var value int64
	var err error
	if strings.HasPrefix(lit, "0x") {
		value, err = strconv.ParseInt(lit[2:], 16, 64) // hexadecimal
	} else if strings.HasPrefix(lit, "0") && len(lit) > 1 {
		value, err = strconv.ParseInt(lit[1:], 8, 64) // octal
	} else {
		value, err = strconv.ParseInt(lit, 10, 64) // decimal
	}
	if err != nil {
		p.setError(NewParserError(ErrorOpts{
			ErrType:       "parse error",
			Message:       fmt.Sprintf("invalid integer: %s", lit),
			File:          p.l.Filename(),
			StartPosition: tok.StartPosition,
			EndPosition:   tok.EndPosition,
			SourceCode:    p.l.GetLineText(tok),
		}))
		return nil
	}
	return ast.NewInt(tok, value)
}

func (p *Parser) parseFloat() ast.Node {
	tok, lit := p.curToken, p.curToken.Literal
	value, err := strconv.ParseFloat(lit, 64)
	if err != nil {
		p.setError(NewParserError(ErrorOpts{
			ErrType:       "parse error",
			Message:       fmt.Sprintf("invalid float: %s", lit),
			File:          p.l.Filename(),
			StartPosition: p.curToken.StartPosition,
			EndPosition:   p.curToken.EndPosition,
			SourceCode:    p.l.GetLineText(p.curToken),
		}))
		return nil
	}
	return ast.NewFloat(tok, value)
}

func (p *Parser) parseSwitch() ast.Node {
	switchToken := p.curToken
	p.nextToken()
	switchValue := p.parseExpression(LOWEST)
	if switchValue == nil {
		return nil
	}
	if !p.expectPeek("switch statement", token.LBRACE) {
		return nil
	}
	p.nextToken()
	p.eatNewlines()
	// Process the switch case statements
	var cases []*ast.Case
	var defaultCaseCount int
	// Each time through this loop we process one case statement
	for !p.curTokenIs(token.RBRACE) {
		if p.curTokenIs(token.EOF) {
			p.setTokenError(p.prevToken, "unterminated switch statement")
			return nil
		}
		if p.curToken.Literal != "case" && p.curToken.Literal != "default" {
			p.setTokenError(p.curToken, "expected 'case' or 'default' (got %s)", p.curToken.Literal)
			return nil
		}
		caseToken := p.curToken
		var isDefaultCase bool
		var caseExprs []ast.Expression
		if p.curTokenIs(token.DEFAULT) {
			isDefaultCase = true
		} else if p.curTokenIs(token.CASE) {
			p.nextToken() // move to the token following "case"
			caseExprs = append(caseExprs, p.parseExpression(LOWEST))
			for p.peekTokenIs(token.COMMA) {
				p.nextToken() // move to the comma
				p.nextToken() // move to the following expression
				caseExprs = append(caseExprs, p.parseExpression(LOWEST))
			}
		} else {
			p.setTokenError(p.curToken, "expected 'case' or 'default' (got %s)", p.curToken.Literal)
			return nil
		}
		if !p.expectPeek("switch statement", token.COLON) {
			return nil
		}
		// Now we are at the block of code to be executed for this case
		p.nextToken()
		p.eatNewlines()
		// An empty case statement is valid
		if p.curTokenIs(token.CASE) || p.curTokenIs(token.DEFAULT) || p.curTokenIs(token.RBRACE) {
			if isDefaultCase {
				defaultCaseCount++
				if defaultCaseCount > 1 {
					p.setTokenError(caseToken, "switch statement has multiple default blocks")
					return nil
				}
				cases = append(cases, ast.NewDefaultCase(caseToken, nil))
			} else {
				cases = append(cases, ast.NewCase(caseToken, caseExprs, nil))
			}
			continue
		}
		blockFirstToken := p.curToken
		var blockStatements []ast.Node
		for {
			// Skip over newlines and semicolons
			for p.curTokenIs(token.NEWLINE) || p.curTokenIs(token.SEMICOLON) {
				if err := p.nextToken(); err != nil {
					return nil
				}
			}
			// Any of these tokens indicate the end of the current case
			if p.curTokenIs(token.CASE) ||
				p.curTokenIs(token.DEFAULT) ||
				p.curTokenIs(token.RBRACE) ||
				p.curTokenIs(token.EOF) {
				break
			}
			// Parse one statement
			if s := p.parseStatement(); s != nil {
				blockStatements = append(blockStatements, s)
			}
			if !p.curTokenIs(token.SEMICOLON) &&
				!statementTerminators[p.peekToken.Type] &&
				!p.peekTokenIs(token.CASE) &&
				!p.peekTokenIs(token.DEFAULT) &&
				!p.peekTokenIs(token.RBRACE) {
				p.peekError("case statement", token.SEMICOLON, p.peekToken)
				return nil
			}
			// Move to the token just beyond the statement
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
		block := ast.NewBlock(blockFirstToken, blockStatements)
		if isDefaultCase {
			defaultCaseCount++
			if defaultCaseCount > 1 {
				p.setTokenError(caseToken, "switch statement has multiple default blocks")
				return nil
			}
			cases = append(cases, ast.NewDefaultCase(caseToken, block))
		} else {
			cases = append(cases, ast.NewCase(caseToken, caseExprs, block))
		}
	}
	return ast.NewSwitch(switchToken, switchValue, cases)
}

func (p *Parser) parseImport() ast.Node {
	importToken := p.curToken
	if !p.expectPeek("an import statement", token.IDENT) {
		return nil
	}
	name := ast.NewIdent(p.curToken)
	var alias *ast.Ident
	if p.peekTokenIs(token.AS) {
		p.nextToken()
		if !p.expectPeek("an import statement", token.IDENT) {
			return nil
		}
		alias = ast.NewIdent(p.curToken)
	}
	return ast.NewImport(importToken, name, alias)
}

func (p *Parser) parseFromImport() ast.Node {
	fromToken := p.curToken
	if !p.expectPeek("a from-import statement", token.IDENT) {
		return nil
	}
	parentModule := make([]*ast.Ident, 0)
	for p.curTokenIs(token.IDENT) {
		parentModule = append(parentModule, ast.NewIdent(p.curToken))
		if err := p.nextToken(); err != nil {
			return nil
		}
		if !p.curTokenIs(token.PERIOD) {
			break
		}
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	if !p.curTokenIs(token.IMPORT) {
		p.setError(NewParserError(ErrorOpts{
			ErrType:       "parse error",
			Message:       "from-import is missing import statement",
			File:          p.l.Filename(),
			StartPosition: p.prevToken.EndPosition,
			EndPosition:   p.prevToken.EndPosition,
			SourceCode:    p.l.GetLineText(p.prevToken),
		}))
		return nil
	}
	importToken := p.curToken
	// If the imports are surrounded by parentheses, we are in a grouped import
	// which may span multiple lines
	isGrouped := false
	if p.peekTokenIs(token.LPAREN) {
		isGrouped = true
		p.nextToken()
		for p.peekTokenIs(token.NEWLINE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
	}
	// Move to the first identifier
	if !p.expectPeek("a from-import statement", token.IDENT) {
		return nil
	}
	var imports []*ast.Import
	for {
		name := ast.NewIdent(p.curToken)
		var alias *ast.Ident
		if p.peekTokenIs(token.AS) {
			p.nextToken()
			if !p.expectPeek("a from-import statement", token.IDENT) {
				return nil
			}
			alias = ast.NewIdent(p.curToken)
		}
		thisImport := ast.NewImport(importToken, name, alias)
		imports = append(imports, thisImport)
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			if isGrouped {
				for p.peekTokenIs(token.NEWLINE) {
					if err := p.nextToken(); err != nil {
						return nil
					}
				}
				if p.peekTokenIs(token.RPAREN) {
					break
				}
			}
			// Advance to the next identifier
			if !p.expectPeek("a from-import statement", token.IDENT) {
				return nil
			}
		} else {
			break
		}
	}
	if isGrouped {
		if !p.expectPeek("a from-import statement", token.RPAREN) {
			return nil
		}
	}
	return ast.NewFromImport(fromToken, parentModule, imports, isGrouped)
}

func (p *Parser) parseBoolean() ast.Node {
	return ast.NewBool(p.curToken, p.curTokenIs(token.TRUE))
}

func (p *Parser) parseNil() ast.Node {
	return ast.NewNil(p.curToken)
}

func (p *Parser) parsePrefixExpr() ast.Node {
	operator := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	right := p.parseExpression(PREFIX)
	if right == nil {
		p.setTokenError(p.curToken, "invalid prefix expression")
		return nil
	}
	return ast.NewPrefix(operator, right)
}

func (p *Parser) parseNewline() ast.Node {
	p.nextToken()
	return nil
}

func (p *Parser) parsePostfix() ast.Statement {
	return ast.NewPostfix(p.prevToken, p.curToken.Literal)
}

func (p *Parser) parseInfixExpr(leftNode ast.Node) ast.Node {
	left, ok := leftNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid expression")
		return nil
	}
	firstToken := p.curToken
	precedence := p.currentPrecedence()
	p.nextToken()
	for p.curTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	right := p.parseExpression(precedence)
	if right == nil {
		p.setTokenError(p.curToken, "invalid expression")
		return nil
	}
	return ast.NewInfix(firstToken, left, firstToken.Literal, right)
}

func (p *Parser) parseTernary(conditionNode ast.Node) ast.Node {
	condition, ok := conditionNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid ternary expression")
		return nil
	}
	if p.tern {
		p.setTokenError(p.curToken, "nested ternary expression detected")
		return nil
	}
	p.tern = true
	defer func() { p.tern = false }()

	firstToken := p.curToken // the "?"
	p.nextToken()            // move past the '?'
	precedence := p.currentPrecedence()
	ifTrue := p.parseExpression(precedence)
	if ifTrue == nil {
		p.setTokenError(p.curToken, "invalid syntax in ternary if true expression")
	}
	if !p.expectPeek("ternary expression", token.COLON) { // moves to the ":"
		return nil
	}
	p.nextToken() // moves after the ":"
	ifFalse := p.parseExpression(precedence)
	if ifFalse == nil {
		p.setTokenError(p.curToken, "invalid syntax in ternary if false expression")
	}
	return ast.NewTernary(firstToken, condition, ifTrue, ifFalse)
}

func (p *Parser) parseGroupedExpr() ast.Node {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek("grouped expression", token.RPAREN) {
		return nil
	}
	return exp
}

// Parses an entire if, else if, else block. Else-ifs are handled recursively.
func (p *Parser) parseIf() ast.Node {
	ifToken := p.curToken
	p.nextToken() // move past the "if"
	cond := p.parseExpression(LOWEST)
	if cond == nil {
		return nil
	}
	if !p.expectPeek("an if expression", token.LBRACE) { // move to the "{"
		return nil
	}
	consequence := p.parseBlock()
	if consequence == nil {
		return nil
	}
	var alternative *ast.Block
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()                // move to the "else"
		if p.peekTokenIs(token.IF) { // this is an "else if"
			p.nextToken() // move to the "if"
			nestedIfToken := p.curToken
			nestedIf := p.parseIf()
			alternative := ast.NewBlock(nestedIfToken, []ast.Node{nestedIf})
			return ast.NewIf(ifToken, cond, consequence, alternative)
		}
		if !p.expectPeek("an if expression", token.LBRACE) {
			return nil
		}
		alternative = p.parseBlock()
		if alternative == nil {
			return nil
		}
	}
	return ast.NewIf(ifToken, cond, consequence, alternative)
}

func (p *Parser) parseFor() ast.Node {
	forToken := p.curToken
	p.nextToken() // Move past 'for'

	// Check for simple form: "for { ... }"
	if p.curTokenIs(token.LBRACE) {
		consequence := p.parseBlock()
		if consequence == nil {
			return nil
		}
		return ast.NewSimpleFor(forToken, consequence)
	}

	// Parse the initialization or condition
	var init, condition, post ast.Node
	if !p.curTokenIs(token.SEMICOLON) {
		init = p.parseStatement()
		if init == nil {
			p.setTokenError(p.curToken, "invalid for loop expression")
			return nil
		}
	}

	// Check for condition-only form: "for condition { ... }"
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		consequence := p.parseBlock()
		if consequence == nil {
			return nil
		}
		return ast.NewFor(forToken, init, consequence, nil, nil)
	}

	// Check if the init statement is a range statement
	var isRangeLoop bool
	switch initNode := init.(type) {
	case *ast.Var:
		_, expr := initNode.Value()
		if _, ok := expr.(*ast.Range); ok {
			isRangeLoop = true
		}
	case *ast.MultiVar:
		_, expr := initNode.Value()
		if _, ok := expr.(*ast.Range); ok {
			isRangeLoop = true
		}
	}

	if isRangeLoop {
		// This is a range-based for loop
		if !p.expectPeek("for range loopx", token.LBRACE) {
			return nil
		}
		consequence := p.parseBlock()
		if consequence == nil {
			return nil
		}
		// Use the init node (which contains the range expression) as the condition
		return ast.NewFor(forToken, init, consequence, nil, nil)
	}

	// If we've reached here, we're dealing with a three-part for loop
	if !p.curTokenIs(token.SEMICOLON) {
		p.setTokenError(p.curToken, "expected semicolon after for loop initialization")
		return nil
	}
	p.nextToken() // Move past the first semicolon

	// Parse the condition
	if !p.curTokenIs(token.SEMICOLON) {
		condition = p.parseExpression(LOWEST)
		if condition == nil {
			p.setTokenError(p.curToken, "invalid for loop condition")
			return nil
		}
	}
	if !p.expectPeek("for loop", token.SEMICOLON) {
		return nil
	}
	p.nextToken() // Move past the second semicolon

	// Parse the post statement
	if !p.curTokenIs(token.LBRACE) {
		if p.curTokenIs(token.IDENT) && (p.peekTokenIs(token.PLUS_PLUS) || p.peekTokenIs(token.MINUS_MINUS)) {
			identToken := p.curToken
			p.nextToken()
			post = ast.NewPostfix(identToken, p.curToken.Literal)
		} else {
			post = p.parseStatement()
		}
		if post == nil {
			p.setTokenError(p.curToken, "invalid for loop post statement")
			return nil
		}
	}

	// Expect the opening brace of the loop body
	if !p.expectPeek("for loop", token.LBRACE) {
		return nil
	}

	consequence := p.parseBlock()
	if consequence == nil {
		return nil
	}

	return ast.NewFor(forToken, condition, consequence, init, post)
}

func (p *Parser) parseBlock() *ast.Block {
	blockToken := p.curToken // Should be '{'
	statements := []ast.Node{}
	if err := p.nextToken(); err != nil { // Move past the '{'
		return nil
	}
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatementStrict()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	if p.curTokenIs(token.EOF) {
		p.setTokenError(blockToken, "unterminated block statement")
		return nil
	}
	return ast.NewBlock(blockToken, statements)
}

func (p *Parser) parseFunc() ast.Node {
	funcToken := p.curToken
	var ident *ast.Ident
	if p.peekTokenIs(token.IDENT) { // Read optional function name
		p.nextToken()
		ident = ast.NewIdent(p.curToken)
	}
	if !p.expectPeek("function", token.LPAREN) { // Move to the "("
		return nil
	}
	defaults, params := p.parseFuncParams()
	if !p.expectPeek("function", token.LBRACE) { // move to the "{"
		return nil
	}
	return ast.NewFunc(funcToken, ident, params, defaults, p.parseBlock())
}

func (p *Parser) parseFuncParams() (map[string]ast.Expression, []*ast.Ident) {
	// If the next parameter is ")", then there are no parameters
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return map[string]ast.Expression{}, nil
	}
	defaults := map[string]ast.Expression{}
	params := make([]*ast.Ident, 0)
	p.nextToken()
	for !p.curTokenIs(token.RPAREN) { // Keep going until we find a ")"
		if p.curTokenIs(token.EOF) {
			p.setTokenError(p.prevToken, "unterminated function parameters")
			return nil, nil
		}
		if !p.curTokenIs(token.IDENT) {
			p.setTokenError(p.curToken, "expected an identifier (got %s)", p.curToken.Literal)
			return nil, nil
		}
		ident := ast.NewIdent(p.curToken)
		params = append(params, ident)
		if err := p.nextToken(); err != nil {
			return nil, nil
		}
		// If there is "=expr" after the name then expr is a default value
		if p.curTokenIs(token.ASSIGN) {
			p.nextToken()
			expr := p.parseExpression(LOWEST)
			if expr == nil {
				return nil, nil
			}
			defaults[ident.String()] = expr
			p.nextToken()
		}
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	return defaults, params
}

func (p *Parser) parseGo() ast.Node {
	goToken := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	if !p.curTokenIs(token.FUNC) && !p.curTokenIs(token.IDENT) {
		p.setTokenError(p.curToken, "invalid go statement")
		return nil
	}
	expr := p.parseExpression(PREFIX)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid go statement")
		return nil
	}
	switch expr := expr.(type) {
	case *ast.Call:
		return ast.NewGo(goToken, expr)
	case *ast.ObjectCall:
		return ast.NewGo(goToken, expr)
	default:
		p.setTokenError(p.curToken, "invalid go statement")
		return nil
	}
}

func (p *Parser) parseDefer() ast.Node {
	deferToken := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	if !p.curTokenIs(token.FUNC) && !p.curTokenIs(token.IDENT) {
		p.setTokenError(p.curToken, "invalid defer statement")
		return nil
	}
	expr := p.parseExpression(PREFIX)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid defer statement")
		return nil
	}
	switch expr := expr.(type) {
	case *ast.Call:
		return ast.NewDefer(deferToken, expr)
	case *ast.ObjectCall:
		return ast.NewDefer(deferToken, expr)
	default:
		p.setTokenError(p.curToken, "invalid defer statement")
		return nil
	}
}

func (p *Parser) parseString() ast.Node {
	strToken := p.curToken
	if strToken.Type == token.BACKTICK || strToken.Type == token.STRING {
		return ast.NewString(strToken)
	}
	if !strings.Contains(strToken.Literal, "{") {
		return ast.NewString(strToken)
	}
	// Template string with interpolation
	tmpl, err := tmpl.Parse(strToken.Literal)
	if err != nil {
		p.setTokenError(strToken, err.Error())
		return nil
	}
	var exprs []ast.Expression
	for _, e := range tmpl.Fragments() {
		if !e.IsVariable() {
			continue
		}
		tmplAst, err := Parse(p.ctx, e.Value())
		if err != nil {
			p.setTokenError(strToken, err.Error())
			return nil
		}
		statements := tmplAst.Statements()
		if len(statements) == 0 {
			exprs = append(exprs, nil)
		} else if len(statements) > 1 {
			p.setTokenError(strToken, "template contains more than one expression")
			return nil
		} else {
			stmt := statements[0]
			expr, ok := stmt.(ast.Expression)
			if !ok {
				p.setTokenError(strToken, "template contains an unexpected statement type")
				return nil
			}
			exprs = append(exprs, expr)
		}
	}
	return ast.NewTemplatedString(strToken, tmpl, exprs)
}

func (p *Parser) parseList() ast.Node {
	bracket := p.curToken
	items := p.parseExprList(token.RBRACKET)
	return ast.NewList(bracket, items)
}

func (p *Parser) parseExprList(end token.Type) []ast.Expression {
	list := make([]ast.Expression, 0)
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	for p.peekTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid syntax in list expression")
		return nil
	}
	list = append(list, expr)
	for p.peekTokenIs(token.COMMA) {
		// move to the comma
		if err := p.nextToken(); err != nil {
			return nil
		}
		// advance across any extra newlines
		for p.peekTokenIs(token.NEWLINE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
		// check if the list has ended after the newlines
		if p.peekTokenIs(end) {
			break
		}
		// move to the next expression
		if err := p.nextToken(); err != nil {
			return nil
		}
		list = append(list, p.parseExpression(LOWEST))
	}
	for p.peekTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	if !p.expectPeek("an expression list", end) {
		return nil
	}
	return list
}

func (p *Parser) parseNodeList(end token.Type) []ast.Node {
	list := make([]ast.Node, 0)
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	for p.peekTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	p.nextToken()
	expr := p.parseNode(LOWEST)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid syntax in list expression")
		return nil
	}
	list = append(list, expr)
	for p.peekTokenIs(token.COMMA) {
		// move to the comma
		if err := p.nextToken(); err != nil {
			return nil
		}
		// advance across any extra newlines
		for p.peekTokenIs(token.NEWLINE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
		// check if the list has ended after the newlines
		if p.peekTokenIs(end) {
			break
		}
		// move to the next expression
		if err := p.nextToken(); err != nil {
			return nil
		}
		list = append(list, p.parseNode(LOWEST))
	}
	for p.peekTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	if !p.expectPeek("a node list", end) {
		return nil
	}
	return list
}

func (p *Parser) parseIndex(leftNode ast.Node) ast.Node {
	left, ok := leftNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid index expression")
		return nil
	}
	indexToken := p.curToken
	var firstIndex, secondIndex ast.Expression
	if !p.peekTokenIs(token.COLON) {
		p.nextToken() // move to the first index
		firstIndex = p.parseExpression(LOWEST)
		if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // move to the "]"
			return ast.NewIndex(indexToken, left, firstIndex)
		}
	}
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // move to the ":"
		if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // move to the "]"
			return ast.NewSlice(indexToken, left, firstIndex, nil)
		}
		p.nextToken() // move to the second index
		secondIndex = p.parseExpression(LOWEST)
	}
	if !p.expectPeek("an index expression", token.RBRACKET) {
		return nil
	}
	return ast.NewSlice(indexToken, left, firstIndex, secondIndex)
}

func (p *Parser) parseAssign(name ast.Node) ast.Node {
	operator := p.curToken
	var ident *ast.Ident
	var index *ast.Index
	switch node := name.(type) {
	case *ast.Ident:
		ident = node
	case *ast.Index:
		index = node
	default:
		p.setTokenError(operator, "unexpected token for assignment: %s", name.Literal())
		return nil
	}
	switch operator.Type {
	case token.PLUS_EQUALS, token.MINUS_EQUALS, token.SLASH_EQUALS,
		token.ASTERISK_EQUALS, token.DECLARE, token.ASSIGN:
		// this is a valid operator
	default:
		p.setTokenError(operator, "unsupported operator for assignment: %s", operator.Literal)
		return nil
	}
	p.nextToken() // move to the RHS value
	right := p.parseExpression(LOWEST)
	if right == nil {
		p.setTokenError(p.curToken, "invalid assignment statement value")
		return nil
	}
	if index != nil {
		return ast.NewAssignIndex(operator, index, right)
	}
	return ast.NewAssign(operator, ident, right)
}

func (p *Parser) parseCall(functionNode ast.Node) ast.Node {
	function, ok := functionNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid call expression")
		return nil
	}
	callToken := p.curToken
	arguments := p.parseNodeList(token.RPAREN)
	if arguments == nil {
		return nil
	}
	return ast.NewCall(callToken, function, arguments)
}

func (p *Parser) parsePipe(firstNode ast.Node) ast.Node {
	first, ok := firstNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid pipe expression")
		return nil
	}
	pipeToken := p.curToken
	exprs := []ast.Expression{first}
	for {
		// Move past the pipe operator itself
		if err := p.nextToken(); err != nil {
			return nil
		}
		// Advance across any extra newlines
		p.eatNewlines()
		// Parse the next expression and add it to the ast.Pipe Arguments
		expr := p.parseExpression(PIPE)
		if expr == nil {
			p.setTokenError(p.curToken, "invalid pipe expression")
			return nil
		}
		exprs = append(exprs, expr)
		// Another pipe character continues the expression
		if p.peekTokenIs(token.PIPE) {
			p.nextToken() // move to the next "|"
			continue
		} else {
			// Anything else indicates the end of the pipe expression
			break
		}
	}
	return ast.NewPipe(pipeToken, exprs)
}

func (p *Parser) parseIn(leftNode ast.Node) ast.Node {
	left, ok := leftNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid in expression")
		return nil
	}
	inToken := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	right := p.parseExpression(PREFIX)
	if right == nil {
		p.setTokenError(p.curToken, "invalid in expression")
		return nil
	}
	return ast.NewIn(inToken, left, right)
}

func (p *Parser) parseRange() ast.Node {
	rangeToken := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	if p.curTokenIs(token.LBRACE) {
		p.setTokenError(p.curToken, "invalid range expression (unexpected \"{\")")
		return nil
	}
	container := p.parseExpression(PREFIX)
	if container == nil {
		p.setTokenError(p.curToken, "invalid range expression")
		return nil
	}
	return ast.NewRange(rangeToken, container)
}

func (p *Parser) parseMapOrSet() ast.Node {
	firstToken := p.curToken
	for p.peekTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return nil
		}
	}
	// Empty {} turns into an empty map (not a set)
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return ast.NewMap(firstToken, nil)
	}
	p.nextToken() // move to the first key
	firstKey := p.parseExpression(LOWEST)
	if p.peekTokenIs(token.COLON) { // This is a map
		p.nextToken() // move to the ":"
		p.nextToken() // move to the first value
		firstValue := p.parseExpression(LOWEST)
		pairs := map[ast.Expression]ast.Expression{firstKey: firstValue}
		for !p.peekTokenIs(token.RBRACE) {
			if p.peekTokenIs(token.NEWLINE) {
				p.nextToken()
				break
			}
			if !p.expectPeek("map", token.COMMA) {
				return nil
			}
			for p.peekTokenIs(token.NEWLINE) {
				if err := p.nextToken(); err != nil {
					return nil
				}
			}
			if p.peekTokenIs(token.RBRACE) {
				break
			}
			key, value := p.parseKeyValue()
			if key == nil || value == nil {
				return nil
			}
			pairs[key] = value
			if !p.peekTokenIs(token.COMMA) {
				break
			}
		}
		for p.peekTokenIs(token.NEWLINE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
		if !p.expectPeek("map", token.RBRACE) {
			return nil
		}
		return ast.NewMap(firstToken, pairs)
	} else { // This is a set
		items := []ast.Expression{firstKey}
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		} else if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			return ast.NewSet(firstToken, items)
		} else {
			p.setTokenError(p.peekToken, "invalid syntax in set expression")
			return nil
		}
		for p.peekTokenIs(token.NEWLINE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
		}
		for !p.peekTokenIs(token.RBRACE) {
			if err := p.nextToken(); err != nil {
				return nil
			}
			key := p.parseExpression(LOWEST)
			items = append(items, key)
			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken() // move to the comma
			for p.peekTokenIs(token.NEWLINE) {
				if err := p.nextToken(); err != nil {
					return nil
				}
			}
		}
		if !p.expectPeek("set", token.RBRACE) {
			return nil
		}
		return ast.NewSet(firstToken, items)
	}
}

func (p *Parser) parseKeyValue() (ast.Expression, ast.Expression) {
	p.nextToken()
	key := p.parseExpression(LOWEST)
	if !p.expectPeek("hash value", token.COLON) {
		return nil, nil
	}
	p.nextToken()
	value := p.parseExpression(LOWEST)
	return key, value
}

func (p *Parser) parseGetAttr(objNode ast.Node) ast.Node {
	obj, ok := objNode.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid attribute expression")
		return nil
	}
	period := p.curToken
	p.nextToken()
	p.eatNewlines()
	if !p.curTokenIs(token.IDENT) {
		p.setTokenError(p.curToken, "expected an identifier after %q", ".")
		return nil
	}
	name := p.parseIdent().(*ast.Ident)
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		callNode := p.parseCall(name)
		call, ok := callNode.(ast.Expression)
		if !ok {
			p.setTokenError(p.curToken, "invalid attribute expression")
			return nil
		}
		return ast.NewObjectCall(period, obj, call)
	} else if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // move to the "="
		p.nextToken() // move to the value
		right := p.parseExpression(LOWEST)
		if right == nil {
			p.setTokenError(p.curToken, "invalid assignment statement value")
			return nil
		}
		return ast.NewSetAttr(obj.Token(), obj, name, right)
	}
	return ast.NewGetAttr(period, obj, name)
}

func (p *Parser) parseSend(channel ast.Node) ast.Node {
	chanExpr, ok := channel.(ast.Expression)
	if !ok {
		p.setTokenError(p.curToken, "invalid send statement channel")
		return nil
	}
	operator := p.curToken
	p.nextToken() // move to the RHS value
	value := p.parseExpression(CALL)
	if value == nil {
		p.setTokenError(operator, "invalid send statement value")
		return nil
	}
	return ast.NewSend(operator, chanExpr, value)
}

func (p *Parser) parseReceive() ast.Node {
	sendToken := p.curToken
	if err := p.nextToken(); err != nil {
		return nil
	}
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		p.setTokenError(p.curToken, "invalid receive statement")
		return nil
	}
	return ast.NewReceive(sendToken, expr)
}

// curTokenIs returns true if the current token has the given type.
func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

// peekTokenIs returns true if the next token has the given type.
func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekToken.Type == t
}

// expectPeek validates if the next token is of the given type, and advances if
// it is. If it's a different type, then an error is stored.
func (p *Parser) expectPeek(context string, t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(context, t, p.peekToken)
	return false
}

// peekPrecedence returns the precedence of the next token.
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// currentPrecedence returns the precedence of the current token.
func (p *Parser) currentPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) eatNewlines() {
	for p.curTokenIs(token.NEWLINE) {
		if err := p.nextToken(); err != nil {
			return
		}
	}
}
