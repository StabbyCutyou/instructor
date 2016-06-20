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
		statement: "o = find(testRecord,\"smedley@gmail.com\")",
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
	{
		statement: "o.CreatedDate",
		results: []Token{
			WORD, PERIOD, WORD, EOF,
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
	CreatedDate time.Time
	Email       string
	OrderIDs    []int
}

func (t *testRecord) Stuff() {}

var testCache = map[string]testRecord{
	"smedley@gmail.com": {Email: "smedley@mail.com", CreatedDate: time.Now(), OrderIDs: []int{1, 3, 4}},
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
		fmt.Println("-----")
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
