package instructor

import (
	"encoding/json"
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
		statement: "o.Dumb.Yes",
		results: []Token{
			WORD, PERIOD, WORD, PERIOD, WORD, EOF,
		},
	},
	{
		statement: "o.Stuff()",
		results: []Token{
			WORD, PERIOD, WORD, LPAREN, RPAREN, EOF,
		},
	},
	{
		statement: "o.Stuff2(false, 50)",
		results: []Token{
			WORD, PERIOD, WORD, LPAREN, WORD, COMMA, WS, WORD, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff()",
		results: []Token{
			WORD, PERIOD, WORD, PERIOD, WORD, LPAREN, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff2(true, 50)",
		results: []Token{
			WORD, PERIOD, WORD, PERIOD, WORD, LPAREN, WORD, COMMA, WS, WORD, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff3(`{\"floops\":5}`)",
		results: []Token{
			WORD, PERIOD, WORD, PERIOD, WORD, LPAREN, WORD, RPAREN, EOF,
		},
	},
}

type testRecord struct {
	CreatedDate time.Time
	Email       string
	OrderIDs    []int
	Dumb        nestedProperty
}

type nestedProperty struct {
	Yes bool
}

type Flooper struct {
	Floops int `json:"floops"`
}

func (n nestedProperty) DeepStuff() int {
	return 30056
}

func (n nestedProperty) DeepStuff2(a bool, b int) int {
	if a {
		return 30056 + b
	}
	return b
}

func (n nestedProperty) DeepStuff3(f Flooper) int {
	return f.Floops
}

func (t *testRecord) Stuff() int {
	return 500001
}

func (t *testRecord) Stuff2(a bool, b int) int {
	if a {
		return 500001 + b
	}
	return b
}

func convertFloop(s string) (interface{}, error) {
	b := []byte(s)
	f := Flooper{}
	if err := json.Unmarshal(b, &f); err != nil {
		fmt.Println(s)
		fmt.Println(err)
		return f, err
	}
	return f, nil
}

var testCache = map[string]*testRecord{
	"smedley@gmail.com": {Email: "smedley@mail.com", CreatedDate: time.Now(), OrderIDs: []int{1, 3, 4}, Dumb: nestedProperty{Yes: true}},
}

func lookup(email string) (interface{}, error) {
	if obj, ok := testCache[email]; ok {
		return obj, nil
	}
	return nil, fmt.Errorf("Not found %s", email)
}

func TestLexerCases(t *testing.T) {
	i := newInterpreter()
	i.RegisterFinder("testRecord", lookup)
	i.RegisterConverter("Flooper", convertFloop)
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

		if err := i.Evaluate(statement); err != nil {
			fmt.Println(err)
			t.Fail()
		}
	}
}
