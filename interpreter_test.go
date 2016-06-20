package instructor

import (
	"strings"
	"testing"
)

func BenchmarkInputHelp(b *testing.B) {
	l := newLexer(strings.NewReader("help"))
	i := NewInterpreter()
	for j := 0; j < b.N; j++ {
		f := l.scan()
		i.Evaluate([]fragment{f})
	}
}
