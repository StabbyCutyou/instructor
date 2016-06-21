package instructor

import (
	"bufio"
	"bytes"
	"io"
)

// The approach in this file was lifted heavily from https://blog.gopheracademy.com/advent-2014/parsers-lexers/

// Token represents a lexical token.
type Token int

// Special tokens
const (
	ILLEGAL Token = iota // 0
	EOF                  // 1: end of input
	WS                   // 2: spaces/tabs
	ASSIGN               // 3: assginment operator, =
	PERIOD               // 4: .
	SQUOTE               // 5: '
	DQUOTE               // 6: "
	COMMA                // 7: ,
	LPAREN               // 8: (
	RPAREN               // 9: )
	WORD                 // 10: anything that isn't a reserved word / token, like properties, function names, variables, or even literals like literal value, like 5, 50.0, or "joseph"
	FIND                 // 11: built in helper for locating structs, hacky
	TICK                 // 12: `
)

const eof = rune(0)

//var validChar = regexp.MustCompile("^[\pL\pN\p{Pc}]*$")

// Scanner is responsible for managing the parsing of a string of input text
type scanner struct {
	r *bufio.Reader
}

type tokenBuffer struct {
	t Token  // last token read
	l string //last literal read
	n int    // size
}

// Lexer is responsible for directly turning the parts of a string of input into tokens and text chunks
type lexer struct {
	s *scanner
	b tokenBuffer
}

// newScanner returns a new instance of the Lexical Scanner
func newScanner(r io.Reader) *scanner {
	return &scanner{
		r: bufio.NewReader(r),
	}
}

// NewLexer returns a new instance of the Parser
func newLexer(r io.Reader) *lexer {
	return &lexer{
		s: newScanner(r),
		b: tokenBuffer{},
	}
}

// read reads the next rune from the buffered reader
// If there is an eof or error, it returns rune(0), eof
func (s *scanner) read() rune {
	c, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return c
}

func (s *scanner) unread() {
	s.r.UnreadRune()
}

// Scan will scan the input text
func (s *scanner) Scan() fragment {
	// Read the next rune
	c := s.read()
	// Return quick if it's already the end of file
	if c == eof {
		return fragment{token: EOF, text: ""}
	}

	if isWhitespace(c) {
		// Rewind by 1, then consume the entire amout of whitespace
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(c) {
		// Rewind by 1, then consume all of the contiguous letters /numbers to later check for a reserved word or variable name, or evaluate as a number
		s.unread()
		return s.scanWord()
	} else if c == '"' {
		// Scan a string which means not ignoring whitespace, until the closing quote
		// We expressly do not unread the double quote beginning the string, and we expressly chuck the double quote closing it
		return s.scanString('"')
	} else if c == '`' {
		// Scan a string which means not ignoring whitespace, until the closing tick
		// We expressly do not unread the tick beginning the string, and we expressly chuck the tick closing it
		// this case is mostly to support idiomatic entry of json
		return s.scanString('`')
	} else if c == '\'' {
		// Scan a rune which means until the closing quote
		// We expressly do not unread the single quote beginning the string, and we expressly chuck the single quote closing it
		return s.scanRune()
	} else if isDigit(c) {
		s.unread()
		return s.scanNumber()
	}

	// Otherwise, see what kind of token it was
	switch c {
	case '=':
		return fragment{token: ASSIGN, text: string(c)}
	case '.':
		return fragment{token: PERIOD, text: string(c)}
	case '(':
		return fragment{token: LPAREN, text: string(c)}
	case ',':
		return fragment{token: COMMA, text: string(c)}
	case ')':
		return fragment{token: RPAREN, text: string(c)}
	default:
		return fragment{token: WORD, text: string(c)}
	}
}

func (s *scanner) scanWhitespace() fragment {
	// Buffer for the current character
	b := bytes.Buffer{}
	b.WriteRune(s.read())
	// Keep going until we find a non whitespace/eol character
	for {
		if c := s.read(); c == eof {
			break
		} else if !isWhitespace(c) {
			// Rewind by one before exitting
			s.unread()
			break
		} else {
			// Straight pass through to the buffer
			b.WriteRune(c)
		}
	}
	return fragment{token: WS, text: b.String()}
}

func (s *scanner) scanNumber() fragment {
	// Buffer for the current character
	b := bytes.Buffer{}
	b.WriteRune(s.read())
	// Keep going until we find something to stop us
	for {
		if c := s.read(); c == eof {
			break
		} else if !isDigit(c) && c != '.' { // Numbers can contain periods, as a special case, for floats/decimals
			// Rewind by one before exitting
			s.unread()
			break
		} else {
			// Straight pass through to the buffer
			b.WriteRune(c)
		}
	}

	return fragment{token: WORD, text: b.String()}
}

func (s *scanner) scanString(boundaryRune rune) fragment {
	// Buffer for the current character
	b := bytes.Buffer{}
	b.WriteRune(s.read())
	// Keep going until we find something to stop us
	for {
		if c := s.read(); c == eof {
			break
		} else if c == boundaryRune { // Break once we encounter a closing quote or tick
			// We are going to chuck the closing quote, so we expressly do not rewind
			break
		} else {
			// Straight pass through to the buffer
			b.WriteRune(c)
		}
	}

	return fragment{token: WORD, text: b.String()}
}

func (s *scanner) scanRune() fragment {
	// Buffer for the current character
	b := bytes.Buffer{}
	b.WriteRune(s.read())
	// Keep going until we find something to stop us
	for {
		if c := s.read(); c == eof {
			break
		} else if c == '\'' { // Break once we encounter a closing quote
			// We are going to chuck the closing quote, so we expressly do not rewind
			break
		} else {
			// Straight pass through to the buffer
			b.WriteRune(c)
		}
	}

	return fragment{token: WORD, text: b.String()}
}

func (s *scanner) scanWord() fragment {
	// Buffer for the current character
	b := bytes.Buffer{}
	b.WriteRune(s.read())
	// Keep going until we find something to stop us
	for {
		if c := s.read(); c == eof {
			break
		} else if !isLetter(c) && !isDigit(c) { // Words can have digits, but they can't start with them - need to enforce this
			// Rewind by one before exitting
			s.unread()
			break
		} else {
			// Straight pass through to the buffer
			b.WriteRune(c)
		}
	}

	// Here is where we'd match a specific reserved word
	word := b.String()
	switch word {
	case "find":
		return fragment{token: FIND, text: word}
	}
	return fragment{token: WORD, text: word}
}

func (l *lexer) scan() fragment {
	// A token has been read, return it
	if l.b.n > 0 {
		l.b.n = 0
		return fragment{token: l.b.t, text: l.b.l}
	}
	// Otherwise, read one from the scanner and send it up
	return l.s.Scan()
}

// unscan pushes the previously read token back onto the buffer.
func (l *lexer) unscan() {
	l.b.n = 1
}

func (l *lexer) scanIgnoreWhitespace() fragment {
	f := l.scan()
	if f.token == WS {
		f = l.scan()
	}
	return f
}

func isWhitespace(c rune) bool {
	return c == ' ' || c == '\t'
}

func isLetter(c rune) bool {
	// Anything a-z, A-Z, or special characters
	// TODO replace this with a proper regex, willya?
	return (c >= 'a' && c < 'z') || (c >= 'A' && c <= 'Z') || c == '*'
}

func isDigit(c rune) bool {
	return (c >= '0' && c <= '9')
}
