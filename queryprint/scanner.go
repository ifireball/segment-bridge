package queryprint

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=tokenType
type tokenType int

const (
	// We set tEOF to be the zero-value. That then also becomes the token type
	// for an zero-value token struct, which ends up being handy when trying to
	// read tokens from a closed channel
	tEOF tokenType = iota
	tAND
	tBANG_EQUAL
	tCOMMA
	tDOT
	tEQUAL
	tEQUAL_EQUAL
	tGREATER
	tGREATER_EQUAL
	tIDENTIFIER
	tLEFT_PAREN
	tLESS
	tLESS_EQUAL
	tLIKE
	tMINUS
	tNOT
	tOR
	tPERCENT
	tPIPE
	tPLUS
	tQUOTED_FIELD
	tRIGHT_PAREN
	tSLASH
	tSPACE
	tSTAR
	tSTRING
	tXOR
)

// Splunk is very strict about identifiers in eval expressions (only alpha-
// numerics and underscores) and very lenient about them anywhere else. Since
// we're not interested in very exact parsing, just enough for pretty printing,
// we're taking the middle ground where an identifier is any sequence that is
// not quoted and it is terminated by any operator character that Splunk
// supports
const identTerms = ",.=>()<-%|+/*! \""

type rtMap map[rune]tokenType

var singleCharTokens = rtMap{
	',': tCOMMA,
	'.': tDOT,
	'=': tEQUAL,
	'>': tGREATER,
	'(': tLEFT_PAREN,
	'<': tLESS,
	'-': tMINUS,
	'%': tPERCENT,
	'|': tPIPE,
	'+': tPLUS,
	')': tRIGHT_PAREN,
	'/': tSLASH,
	'*': tSTAR,
}
var doubleCharTokens = map[rune]rtMap{
	'!': {'=': tBANG_EQUAL},
	'=': {'=': tEQUAL_EQUAL},
	'>': {'=': tGREATER_EQUAL},
	'<': {'=': tLESS_EQUAL},
}
var keyWords = map[string]tokenType{
	"AND": tAND,
	"NOT": tNOT,
	"OR":  tOR,
	"XOR": tXOR,
}

type token struct {
	typ   tokenType
	value string
}

func (t token) String() string {
	if t.value == "" {
		return fmt.Sprintf("token{typ: %v}", t.typ)
	}
	return fmt.Sprintf("token{typ: %v, value: %#v}", t.typ, t.value)
}

type scanner struct {
	in      io.RuneScanner
	out     chan<- token
	current rune
	err     error
}

func ScanQuery(in io.RuneScanner, out chan<- token) error {
	p := scanner{in: in, out: out}
	return p.parse()
}

func (p *scanner) parse() error {
	for p.advance() {
		switch p.current {
		case '\\':
			p.identifier()
		case '"':
			p.string()
		case '\'':
			p.quotedField()
		default:
			if !(p.tryDoubleCharToken() ||
				p.trySingleCharToken() ||
				p.trySpace()) {
				p.identifier()
			}
		}
	}
	return p.err
}

func (p *scanner) trySingleCharToken() bool {
	if typ, ok := singleCharTokens[p.current]; ok {
		p.out <- token{typ: typ}
		return true
	}
	return false
}

func (p *scanner) tryDoubleCharToken() bool {
	if nextChars, ok := doubleCharTokens[p.current]; ok {
		oldCurrent := p.current
		if p.advance() {
			if typ, ok := nextChars[p.current]; ok {
				p.out <- token{typ: typ}
				return true
			}
			p.stepBack()
			// We need to do this because stepBack does not usually
			// restore p.current
			p.current = oldCurrent
		}
	}
	return false
}

func (p *scanner) trySpace() bool {
	if !unicode.IsSpace(p.current) {
		return false
	}
	for p.advance() {
		if !unicode.IsSpace(p.current) {
			p.stepBack()
			break
		}
	}
	p.out <- token{typ: tSPACE}
	return true
}

func (p *scanner) identifier() {
	var tokenVal strings.Builder
	// Go beck to starting rune so we can parse it
	if !p.stepBack() {
		return
	}
	for p.advance() {
		if p.current == '\\' {
			// Because we're only pretty printing, we're leaving backslashes
			// as they are, rather then trying to parse them, but we do make
			// sure to not terminate on escaped special characters.
			tokenVal.WriteRune('\\')
			if !p.advance() {
				break
			}
		} else if strings.ContainsRune(identTerms, p.current) || unicode.IsSpace(p.current) {
			p.stepBack()
			break
		}
		tokenVal.WriteRune(p.current)
	}
	var tokenStr = tokenVal.String()
	if typ, ok := keyWords[tokenStr]; ok {
		p.out <- token{typ: typ}
	} else {
		p.out <- token{typ: tIDENTIFIER, value: tokenVal.String()}
	}
}

func (p *scanner) string() {
	p.quotedToken(tSTRING, '"')
}

func (p *scanner) quotedField() {
	p.quotedToken(tQUOTED_FIELD, '\'')
}

func (p *scanner) quotedToken(typ tokenType, terminator rune) {
	var tokenVal strings.Builder
	// Could check here that current == terminator but we're just going to
	// assume we got the switch in parse() right
	tokenVal.WriteRune(p.current)
	for p.advance() {
		tokenVal.WriteRune(p.current)
		if p.current == '\\' {
			// Because we're only pretty printing, we're leaving backslashes
			// as they are, rather then trying to parse them, but we do make
			// sure to not terminate on escaped special characters.
			if !p.advance() {
				break
			}
			tokenVal.WriteRune(p.current)
		} else if p.current == terminator {
			break
		}
	}
	// We could detect unterminated string here by ensuring that
	// t.current == terminator, but since we're only pretty-printing we're not
	// going to bother doing that
	p.out <- token{typ: typ, value: tokenVal.String()}
}

func (p *scanner) advance() bool {
	if p.err != nil {
		return false
	}
	r, _, err := p.in.ReadRune()
	if err != nil {
		if err != io.EOF {
			p.err = err
		}
		return false
	}
	p.current = r
	return true
}

func (p *scanner) stepBack() bool {
	if p.err != nil {
		return false
	}
	if err := p.in.UnreadRune(); err != nil {
		p.err = err
		return false
	}
	return true
}
