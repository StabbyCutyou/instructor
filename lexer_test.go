package instructor

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type LexerTestCase struct {
	statement string
	results   []Token
}

var cases = []LexerTestCase{
	{
		statement: "o = find(testRecord,smedley)",
		results: []Token{
			WORD, WS, ASSIGN, WS, FIND, LPAREN, WORD, COMMA, WORD, RPAREN, EOF,
		},
	},
	{
		statement: "o",
		results: []Token{
			WORD, EOF,
		},
	},
	//{
	//	statement: "o.Stuff(100.0 float64, \"joe shmoe\" string, 'X' rune, 50 *int, true bool)",
	//	results: []Token{
	//		WORD, PERIOD, WORD, LPAREN, WORD, WS, WORD, COMMA, WS, WORD, WS, WORD, COMMA, WS, WORD, WS, WORD, COMMA, WS, WORD, WS, WORD, COMMA, WS, WORD, WS, WORD, RPAREN, EOF,
	//	},
	//},
	//
	//{
	//	statement: "o.Name",
	//	results: []Token{
	//		WORD, PERIOD, WORD, EOF,
	//	},
	//},
}

type testRecord struct {
	createdDate time.Time
	email       string
	orderIDs    []int
}

var testCache = map[string]testRecord{
	"smedley": {email: "smedley@mail.com", createdDate: time.Now(), orderIDs: []int{1, 3, 4}},
}

func lookup(email string) (interface{}, error) {
	if obj, ok := testCache[email]; ok {
		return obj, nil
	}
	return nil, fmt.Errorf("Not found %s", email)
}

func TestLexerCases(t *testing.T) {
	i := NewInterpreter()
	i.RegisterFinder("testRecord", lookup)
	for _, c := range cases {
		r := strings.NewReader(c.statement)
		p := newLexer(r)

		var f fragment
		var k = 0
		statement := make([]fragment, 0)
		for f.token != EOF {
			f = p.scan()
			if f.token != c.results[k] {
				fmt.Printf("Token:%d String:%s, C:%s\n", f.token, f.text, c.results[k])
				t.Fail()
			}
			statement = append(statement, f)
			k++
		}

		fmt.Println(i.Evaluate(statement))
	}
}
