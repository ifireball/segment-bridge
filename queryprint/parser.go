package queryprint

import "errors"

type parser struct {
	in <-chan token
	current token
	err error
	stop bool
}

func ParseQuery(in <-chan token) (query, error) {
	p := parser{in: in}
	p.advance()
	return p.query(), p.err
}

// s -> tSPACE
// query -> command ("|" command)*
func (p *parser) query() (q query) {
	cmd := p.command()
	for !p.stop {
		q = append(q, cmd)
		if !p.advanceIf(tPIPE) {
			break
		}
		cmd = p.command()
	}
	if !p.stop {
		p.error("Found unexpected tokens following query")
	}
	return q
}

// command -> evalCmd | fieldsCmd | searchCmd
func (p *parser) command() (cmd command) {
	if cmdTok, ok := p.expect(tIDENTIFIER,
		"Expected Splunk command after '|' and in query beginning"); ok {
			switch cmdTok.value {
			case "eval": cmd = p.evalCmd()
			case "fields": cmd = p.fieldsCmd()
			default: cmd = p.searchCmd()
			}
		}
	// Return empty struct on failed expect, calling code should be ignoring it
	// anyway because we should be in EOF mode already
	return
}

// evalCmd -> "eval" (s? evalExpr ("," s? evalExpr)*)?
func (p *parser) evalCmd() command {
	var cmdArgs commaSepArgs
	for !p.stop {
		p.advanceIf(tSPACE)
		if p.currentIs(tPIPE) {
			break
		}
		expr := p.evalExpr()
		cmdArgs = append(cmdArgs, expr)
		// evalExpr should read everything up to the comma
		p.advanceIf(tCOMMA)
	}
	return command{"eval", cmdArgs}
}

// fieldsCmd -> "fields" s? (("+"|"-") s)? commaSepField (s? "," s? commaSepField)*
// searchCmd -> tIDENTIFIER s spcSepField (s spcSepField) *

// evalExpr -> evalExprElem s? (evalExprElem s?)*
// evalExprElem => (callExpr|parenExpr|plainExpr)
// callExpr -> tIDENTIFIER s? "(" s? evalExpr ")"
// parenExpr -> "(" s? evalExpr ")"
// plainExpr -> (^ ( "(" | ")" | "," | "|" ) )
func (p *parser) evalExpr() (elements exprElements) {
	for !(p.stop || p.currentIs(tCOMMA) || p.currentIs(tPIPE) || p.currentIs(tRIGHT_PAREN)) {
		if p.advanceIf(tLEFT_PAREN) {
			p.advanceIf(tSPACE)
			parenExpr := p.evalExpr()
			if p.advanceIf(tRIGHT_PAREN) {
				p.error("Found unbalanced parentheses")
				return
			}
			if len(parenExpr) <= 0 {
				p.error("Expected expression inside parentheses")
				return
			}
		} else {
			
		}
	}
	return
}

// commaSepField -> (^ "," s) (s (^ "," s))*
// spcSepField -> (^ s) (^ s)*


func (p *parser) expect(typ tokenType, msg string) (t token, r bool) {
	t = p.current
	r = p.advanceIf(typ)
	if !r {
		t = token{}
		// We bail out of parsing the rest of the tokens on error
		p.error(msg)
	}
	return
}

func (p *parser) error(msg string) {
	if p.err != nil {
		// Only report the 1st error we find
		return
	}
	p.err = errors.New(msg)
	// Exhaust the in channel to unblock any routine that may be trying to
	// write to it and put the parser in EOF mode.
	for p.advance() {}
}

func (p *parser) advanceIf(typ tokenType) bool {
	if p.currentIs(typ) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) currentIs(typ tokenType) bool {
	return p.current.typ == typ
}

func (p *parser) advance() bool {
	if !p.stop {
		p.current = <-p.in
		if p.current.typ == tEOF {
			p.stop = true
		}
	}
	return !p.stop
}
