package processors

import (
	"context"
	"fmt"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/sreagent/sreagent/internal/engine/pipeline"
	"github.com/sreagent/sreagent/internal/model"
)

func init() {
	pipeline.Register("event_drop", newEventDrop)
}

// eventDropProcessor drops events when a safe expression evaluates to true.
type eventDropProcessor struct {
	Condition string `json:"condition"` // safe expression (no arbitrary code execution)
}

func newEventDrop(config map[string]interface{}) (pipeline.Processor, error) {
	p := &eventDropProcessor{}
	if v, ok := config["condition"].(string); ok {
		p.Condition = v
	}
	if p.Condition == "" {
		return nil, fmt.Errorf("event_drop: condition is required")
	}
	// Validate the expression parses at config time
	if _, err := parseExpr(p.Condition); err != nil {
		return nil, fmt.Errorf("event_drop: invalid condition: %w", err)
	}
	return p, nil
}

func (p *eventDropProcessor) Process(ctx context.Context, event *model.AlertEvent) (*model.AlertEvent, string, error) {
	// Build template data from event
	data := map[string]interface{}{
		"AlertName":   event.AlertName,
		"Severity":    string(event.Severity),
		"Status":      string(event.Status),
		"Labels":      map[string]string(event.Labels),
		"Annotations": map[string]string(event.Annotations),
		"Source":      event.Source,
	}

	result, err := evalExpr(p.Condition, data)
	if err != nil {
		return event, "", fmt.Errorf("event_drop: evaluation failed: %w", err)
	}

	if result {
		return nil, "event_drop: condition matched, event dropped", nil
	}
	return event, "event_drop: condition not matched, event kept", nil
}

// ---------------------------------------------------------------------------
// Safe expression evaluator — NO arbitrary code execution.
//
// Grammar:
//   expr   = andExpr ("OR" andExpr)*
//   andExpr = unary ("AND" unary)*
//   unary  = "NOT" unary | "(" expr ")" | comparison
//   comparison = field ("==" | "!=") stringLit
//   field  = "." ident ("." ident)*
//   stringLit = '...' | "..."
//
// Example: .Labels.env == "prod" AND .Severity != "low"
// ---------------------------------------------------------------------------

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokDot     // .
	tokEq      // ==
	tokNeq     // !=
	tokLParen  // (
	tokRParen  // )
	tokNot     // NOT
	tokAnd     // AND
	tokOr      // OR
)

type token struct {
	kind tokenKind
	val  string
}

type exprLexer struct {
	s   scanner.Scanner
	cur token
}

func newLexer(input string) *exprLexer {
	l := &exprLexer{}
	l.s.Init(strings.NewReader(input))
	l.s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings
	l.s.IsIdentRune = func(ch rune, i int) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}
	l.next()
	return l
}

func (l *exprLexer) next() {
	ch := l.s.Scan()
	text := l.s.TokenText()

	switch ch {
	case scanner.EOF:
		l.cur = token{kind: tokEOF}
	case '.':
		l.cur = token{kind: tokDot, val: "."}
	case '(':
		l.cur = token{kind: tokLParen, val: "("}
	case ')':
		l.cur = token{kind: tokRParen, val: ")"}
	case '=':
		// peek for ==
		if l.s.Peek() == '=' {
			l.s.Scan()
			l.cur = token{kind: tokEq, val: "=="}
		} else {
			l.cur = token{kind: tokEOF} // invalid, will error later
		}
	case '!':
		if l.s.Peek() == '=' {
			l.s.Scan()
			l.cur = token{kind: tokNeq, val: "!="}
		} else {
			l.cur = token{kind: tokNot, val: "!"}
		}
	default:
		upper := strings.ToUpper(text)
		switch upper {
		case "AND", "&&":
			l.cur = token{kind: tokAnd, val: "AND"}
		case "OR", "||":
			l.cur = token{kind: tokOr, val: "OR"}
		case "NOT":
			l.cur = token{kind: tokNot, val: "NOT"}
		default:
			// Handle string literals that scanner returns as ident (single-quoted)
			if (strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'")) ||
				(strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"")) {
				l.cur = token{kind: tokString, val: text[1 : len(text)-1]}
			} else {
				l.cur = token{kind: tokIdent, val: text}
			}
		}
	}
}

// exprNode is an AST node for the safe expression language.
type exprNode interface {
	eval(data map[string]interface{}) (bool, error)
}

// compNode represents a comparison like .Labels.env == "prod".
type compNode struct {
	field    string // dot-path, e.g. ".Labels.env"
	op       string // "==" or "!="
	expected string
}

func (n *compNode) eval(data map[string]interface{}) (bool, error) {
	actual, err := resolveField(n.field, data)
	if err != nil {
		return false, err
	}
	switch n.op {
	case "==":
		return actual == n.expected, nil
	case "!=":
		return actual != n.expected, nil
	default:
		return false, fmt.Errorf("unsupported operator %q", n.op)
	}
}

// andNode represents A AND B.
type andNode struct {
	left, right exprNode
}

func (n *andNode) eval(data map[string]interface{}) (bool, error) {
	l, err := n.left.eval(data)
	if err != nil {
		return false, err
	}
	if !l {
		return false, nil
	}
	return n.right.eval(data)
}

// orNode represents A OR B.
type orNode struct {
	left, right exprNode
}

func (n *orNode) eval(data map[string]interface{}) (bool, error) {
	l, err := n.left.eval(data)
	if err != nil {
		return false, err
	}
	if l {
		return true, nil
	}
	return n.right.eval(data)
}

// notNode represents NOT A.
type notNode struct {
	inner exprNode
}

func (n *notNode) eval(data map[string]interface{}) (bool, error) {
	v, err := n.inner.eval(data)
	if err != nil {
		return false, err
	}
	return !v, nil
}

// alwaysTrueNode is a boolean literal.
type alwaysTrueNode struct{ val bool }

func (n *alwaysTrueNode) eval(data map[string]interface{}) (bool, error) {
	return n.val, nil
}

// parseExpr is the entry point for parsing a safe expression string.
func parseExpr(input string) (exprNode, error) {
	l := newLexer(input)
	node, err := parseOr(l)
	if err != nil {
		return nil, err
	}
	if l.cur.kind != tokEOF {
		return nil, fmt.Errorf("unexpected token %q at end of expression", l.cur.val)
	}
	return node, nil
}

func parseOr(l *exprLexer) (exprNode, error) {
	left, err := parseAnd(l)
	if err != nil {
		return nil, err
	}
	for l.cur.kind == tokOr {
		l.next()
		right, err := parseAnd(l)
		if err != nil {
			return nil, err
		}
		left = &orNode{left: left, right: right}
	}
	return left, nil
}

func parseAnd(l *exprLexer) (exprNode, error) {
	left, err := parseUnary(l)
	if err != nil {
		return nil, err
	}
	for l.cur.kind == tokAnd {
		l.next()
		right, err := parseUnary(l)
		if err != nil {
			return nil, err
		}
		left = &andNode{left: left, right: right}
	}
	return left, nil
}

func parseUnary(l *exprLexer) (exprNode, error) {
	if l.cur.kind == tokNot {
		l.next()
		inner, err := parseUnary(l)
		if err != nil {
			return nil, err
		}
		return &notNode{inner: inner}, nil
	}
	return parsePrimary(l)
}

func parsePrimary(l *exprLexer) (exprNode, error) {
	switch l.cur.kind {
	case tokLParen:
		l.next()
		node, err := parseOr(l)
		if err != nil {
			return nil, err
		}
		if l.cur.kind != tokRParen {
			return nil, fmt.Errorf("expected ')', got %q", l.cur.val)
		}
		l.next()
		return node, nil

	case tokDot:
		// Parse field path
		fieldPath, err := parseFieldPath(l)
		if err != nil {
			return nil, err
		}
		// Expect operator
		if l.cur.kind != tokEq && l.cur.kind != tokNeq {
			return nil, fmt.Errorf("expected '==' or '!=' after field %q, got %q", fieldPath, l.cur.val)
		}
		op := l.cur.val
		l.next()
		// Expect string literal
		if l.cur.kind != tokString {
			return nil, fmt.Errorf("expected string literal after %q, got %q", op, l.cur.val)
		}
		expected := l.cur.val
		l.next()
		return &compNode{field: fieldPath, op: op, expected: expected}, nil

	case tokIdent:
		// "true" or "false" literal
		upper := strings.ToUpper(l.cur.val)
		if upper == "TRUE" {
			l.next()
			return &alwaysTrueNode{val: true}, nil
		}
		if upper == "FALSE" {
			l.next()
			return &alwaysTrueNode{val: false}, nil
		}
		return nil, fmt.Errorf("unexpected identifier %q", l.cur.val)

	default:
		return nil, fmt.Errorf("unexpected token %q", l.cur.val)
	}
}

// parseFieldPath parses a dot-separated field path like .Labels.env
func parseFieldPath(l *exprLexer) (string, error) {
	if l.cur.kind != tokDot {
		return "", fmt.Errorf("expected field path starting with '.', got %q", l.cur.val)
	}
	var parts []string
	for l.cur.kind == tokDot {
		l.next()
		if l.cur.kind != tokIdent {
			return "", fmt.Errorf("expected identifier after '.', got %q", l.cur.val)
		}
		parts = append(parts, l.cur.val)
		l.next()
	}
	return "." + strings.Join(parts, "."), nil
}

// resolveField walks the data map using a dot-path like ".Labels.env".
func resolveField(path string, data map[string]interface{}) (string, error) {
	if !strings.HasPrefix(path, ".") {
		return "", fmt.Errorf("field path must start with '.'")
	}
	parts := strings.Split(path[1:], ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			// Try map[string]string
			ms, ok2 := current.(map[string]string)
			if !ok2 {
				return "", fmt.Errorf("cannot access %q: not a map", part)
			}
			v, ok3 := ms[part]
			if !ok3 {
				return "", nil // field not found, return empty string
			}
			return v, nil
		}
		current = m[part]
	}
	if current == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", current), nil
}

// evalExpr parses and evaluates a safe expression against event data.
func evalExpr(input string, data map[string]interface{}) (bool, error) {
	node, err := parseExpr(input)
	if err != nil {
		return false, err
	}
	return node.eval(data)
}
