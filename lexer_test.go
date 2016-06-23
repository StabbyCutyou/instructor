package instructor

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			VARIABLE, WS, ASSIGN, WS, FIND, LPAREN, VARIABLE, COMMA, STRING, RPAREN, EOF,
		},
	},
	{
		statement: "o",
		results: []Token{
			VARIABLE, EOF,
		},
	},
	{
		statement: "o.Dumb.Yes",
		results: []Token{
			VARIABLE, FIELD, FIELD, EOF,
		},
	},
	{
		statement: "o.Stuff()",
		results: []Token{
			VARIABLE, FIELD, LPAREN, RPAREN, EOF,
		},
	},
	{
		statement: "o.Stuff2(false, 50)",
		results: []Token{
			VARIABLE, FIELD, LPAREN, BOOL, COMMA, WS, NUMBER, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff()",
		results: []Token{
			VARIABLE, FIELD, FIELD, LPAREN, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff2(true, 50)",
		results: []Token{
			VARIABLE, FIELD, FIELD, LPAREN, BOOL, COMMA, WS, NUMBER, RPAREN, EOF,
		},
	},
	{
		statement: "o.Dumb.DeepStuff3(`{\"floops\":5}`)",
		results: []Token{
			VARIABLE, FIELD, FIELD, LPAREN, STRING, RPAREN, EOF,
		},
	},
	{
		statement: "o.Orders[1].CustomID(true)",
		results: []Token{
			VARIABLE, FIELD, LBRACK, NUMBER, RBRACK, FIELD, LPAREN, BOOL, RPAREN, EOF,
		},
	},
}

type testRecord struct {
	CreatedDate time.Time
	Email       string
	Orders      []*Order
	Dumb        nestedProperty
}

type nestedProperty struct {
	Yes bool
}

type Flooper struct {
	Floops int `json:"floops"`
}

type Order struct {
	ID        string
	NumFloops int
}

func (o *Order) CustomID(addfloops bool) string {
	if addfloops {
		return "onum-" + o.ID + "-" + strconv.Itoa(o.NumFloops)
	}
	return "onum-" + o.ID
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
	"smedley@gmail.com": {Email: "smedley@mail.com", CreatedDate: time.Now(), Orders: []*Order{{ID: "xxx", NumFloops: 10}, {ID: "rrr", NumFloops: 5}, {ID: "yyy", NumFloops: 15}}, Dumb: nestedProperty{Yes: true}},
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
				fmt.Printf("Got Token:%d String:%s, Expected Token:%d\n", f.token, f.text, c.results[k])
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
