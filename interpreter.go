package instructor

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// interpreter is a quasi-runtime that holds objects in memory, knows how to find and convert things
// and interprets statements
type interpreter struct {
	finders    finders
	converters converters
	heap       heap
}

// newInterpreter returns a new Instructor
func newInterpreter() *interpreter {
	return &interpreter{
		finders: make(finders),
		heap:    make(heap),
		converters: map[string]Converter{
			"bool":     stringToBool,
			"*bool":    stringToBool,
			"int":      stringToInt,
			"*int":     stringToInt,
			"uint":     stringToUint,
			"*uint":    stringToPUint,
			"float64":  stringToFloat64,
			"*float64": stringToFloat64,
			"string":   stringToString,
			"*string":  stringToString,
		},
	}
}

type statementType int

const (
	INVALID      = iota // 0
	ASSIGNMENT          // 1
	METHODCALL          // 2
	PROPERTYCALL        // 3
	INSPECT             // 4
	LOOKUP              // 5
)

type preparedStatement struct {
	fullStatement statement
	lhs           statement
	rhs           statement
	t             statementType
}

func (i *interpreter) prepareStatement(s statement) (preparedStatement, error) {
	// First thing, lets just make sure no pesky whitespace is hanging around
	s = cleanWhitespace(s)
	ps := preparedStatement{fullStatement: s}
	// Fail fast if it's a simple call to find
	if s[0].token == FIND {
		ps.t = LOOKUP
		ps.lhs = s
		return ps, nil
	}
	// Fail fast if it's just inspecting a variable
	if len(s) == 2 && s[0].token == WORD {
		ps.t = INSPECT
		ps.lhs = s
		return ps, nil
	}
	for i := 0; i < len(s); i++ {
		f := s[i]
		// If there is a singleequal anywhere, this puts us into a lhs = rhs assignment
		if f.token == ASSIGN {
			ps.t = ASSIGNMENT
			ps.lhs = s[0:i]  // Everything on the left of the = is LHS
			ps.rhs = s[i+1:] // Everything +1 the current point is the RHS
			return ps, nil
		} else if f.token == LPAREN {
			// If we haven't hit an assignment but there is a ( somewhere, this is a direct method invocation
			ps.t = METHODCALL
			ps.lhs = s
			return ps, nil
		}
	}
	// If it wasn't a variable inspection, and there were no = or ( found, it must be a property invocation
	ps.t = PROPERTYCALL
	ps.lhs = s
	return ps, nil
}

// Evaluate is a set of rules dictating how the tokens will be interpreted.
func (i *interpreter) Evaluate(s statement) error {
	obj, err := i.evaluateStatement(s)
	if err != nil {
		return err
	}
	spew.Dump(obj)
	return nil
}

// Evaluate is a set of rules dictating how the tokens will be interpreted.
func (i *interpreter) evaluateStatement(s statement) (interface{}, error) {
	ps, err := i.prepareStatement(s)
	if err != nil {
		return nil, err
	}
	switch ps.t {
	case INSPECT:
		obj, err := i.lookupVariable(ps.lhs)
		if err != nil {
			return nil, err
		}
		return obj, nil
	case LOOKUP:
		// Pass from the 4th token on (LPAREN <-> RPAREN : EOF))
		// get back arguments for dynamic finder
		stype, id, err := statementToFindArgs(ps.lhs[1:])
		if err != nil {
			return nil, err
		}
		obj, err := i.find(stype, id)
		if err != nil {
			return nil, err
		}
		return obj, nil
	case METHODCALL:
		chain, args := statementToInvocationChainAndParams(ps.lhs)
		// It's a method invocation
		// The first word in the statement is the variable
		// the next N are the property chains
		// the args are the things to invoke on the method
		obj, err := i.callMethodChain(chain, args)
		if err != nil {
			return nil, err
		}
		return obj, nil
	case PROPERTYCALL:
		chain, _ := statementToInvocationChainAndParams(ps.lhs)
		// It's a property invocation
		// The first word in the statement is the variable
		// the next N are the property chains
		obj, err := i.callPropertyChain(chain)
		if err != nil {
			return nil, err
		}
		return obj, nil
	case ASSIGNMENT:
		// This is the remaining case to port over
		// it's still in the lookup branch below
		// we need to do some recursion here in the new world order
		// send LHS and RHS down the Evaluate tree, and get back their objects
		// then assign RHS to LHS.
		// This requires re-designing Evaluate to return an interface{}, error
		// likely - break this into 2 methods. A wrapper not to be recursed by the
		// caller, and the actual method, which returns an object and an error, for
		// the purposes of being able to be called recursively?
		lhs, err := i.evaluateStatement(ps.lhs)
		initVar := false
		if err != nil && strings.Contains(err.Error(), "Unknown variable") {
			// The left hand side is a variable call, but it hasn't been defined yet
			// we're setting it for the first time
			initVar = true
		} else if err != nil {
			return nil, err
		}
		rhs, err := i.evaluateStatement(ps.rhs)
		if err != nil {
			return nil, err
		}
		if initVar {
			// LHS was a variable that didn't exist
			i.storeInHeap(ps.lhs[0].text, rhs)
			lhs = rhs
		} else {
			// LHS is either an existing variable
			// or
			// a property on something else
			vlhs := reflect.ValueOf(lhs)
			vlhs.Set(reflect.ValueOf(rhs))
		}

		return lhs, nil
	case INVALID:
	default:
		return nil, fmt.Errorf("Error: \"%s\" is not a valid statement", ps.fullStatement)
	}
	return nil, nil
}

func (i *interpreter) callMethodChain(chain statement, args statement) ([]interface{}, error) {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panicxxxx: %s\n", err)
		}
	}()
	var err error
	max := len(chain)
	// Get the object to call the method on
	// if max-1 is the last element, and that is the funcion,
	var obj interface{}
	var ok bool
	if max > 2 {
		// if max > 2, then the chain needs to be evaluated to get the object
		// to call the method on. Otherwise, the 2nd fragment is the method, and
		// it's called directly
		obj, err = i.crawlPropertyChain(chain[:max-1])
		if err != nil {
			return nil, err
		}
	} else {
		obj, ok = i.heap[chain[0].text]
		if !ok {
			return nil, fmt.Errorf("Error: Unknown variable %s", chain[0].text)
		}
	}

	// Get the reflect value and look up the method
	v := reflect.ValueOf(obj)
	// Don't do this for methods.. but perhaps we need a ptr/nonptr fallback?
	//if v.Kind() == reflect.Ptr {
	//	v = v.Elem()
	//}
	mname := chain[max-1].text
	m := v.MethodByName(mname)
	mtype := m.Type()

	inputArgs, err := i.statementToArgs(mtype, args)
	if err != nil {
		return nil, err
	}

	// Call the Method with the value args
	r := m.Call(inputArgs)
	results := make([]interface{}, len(r))
	// Print all the results
	for i, rv := range r {
		fmt.Printf("%s[%d] : %v\n", mname, i, rv)
		results[i] = rv.Interface()
	}
	return results, nil
}

func (i *interpreter) crawlPropertyChain(statement statement) (interface{}, error) {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panicxx: %s\n", err)
		}
	}()
	obj, ok := i.heap[statement[0].text]
	if !ok {
		return nil, fmt.Errorf("Error: Unknown variable %s", statement[0].text)
	}
	currentVal := reflect.ValueOf(obj)
	parsingIndex := false
	for _, f := range statement[1:] {
		if f.token != EOF && f.token != PERIOD && f.token != RBRACK {
			if f.token == LBRACK {
				parsingIndex = true
			} else if parsingIndex {
				indexval, err := strconv.Atoi(f.text)
				if err != nil {
					return nil, fmt.Errorf("Error: Unable to use %s as an index value for %v. Original error: %s", f.text, currentVal, err.Error())
				}
				currentVal = currentVal.Index(indexval)
				parsingIndex = false
			} else {
				// We're not dealing with an indexing operation, this is a straight invocation of a property
				// Deref if we're dealing with a pointer
				if currentVal.Kind() == reflect.Ptr {
					currentVal = currentVal.Elem()
				}
				p := currentVal.FieldByName(f.text)
				currentVal = p
			}
		}
	}

	return currentVal.Interface(), nil
}

func (i *interpreter) callPropertyChain(statement statement) (interface{}, error) {
	obj, err := i.crawlPropertyChain(statement)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", obj)
	return obj, nil
}

func statementToInvocationChainAndParams(s statement) (statement, statement) {
	chain := make(statement, 0)
	args := make(statement, 0)
	hitParen := false
	for _, f := range s {
		if f.token == LPAREN {
			hitParen = true
			args = append(args, f)
		} else if !hitParen && f.token != PERIOD {
			// We're still in the chain
			chain = append(chain, f)
		} else if hitParen {
			// We're in the method params
			args = append(args, f)
		}
	}
	if hitParen {
		return chain, args
	}
	return chain, nil
}

func statementToFindArgs(s statement) (string, string, error) {
	max := len(s)
	if max == 6 && // Find statement should only have 7 fragements
		s[0].token == LPAREN &&
		s[max-1].token == EOF &&
		s[max-2].token == RPAREN {
		// Valid set of args so far
		return s[1].text, s[3].text, nil
	}
	// TODO re-assemble statement for error message
	return "", "", fmt.Errorf("Error: Invalid set of arguments for Find")
}

func cleanWhitespace(s statement) statement {
	results := make(statement, 0)
	for _, f := range s {
		if f.token != WS {
			results = append(results, f)
		}
	}
	return results
}

// RegisterFinder is for registering one of your custom finders to look up your structs
func (i *interpreter) RegisterFinder(name string, f Finder) {
	i.finders[name] = f
}

// RegisterConverter is for registering one of your custom converters to convert cli arguments to typed values
func (i *interpreter) RegisterConverter(name string, c Converter) {
	i.converters[name] = c
}

func (i *interpreter) storeInHeap(id string, obj interface{}) error {
	// Store record in i.instances
	i.heap[id] = obj

	return nil
}

func (i *interpreter) lookupVariable(s statement) (interface{}, error) {
	f := s[0]
	obj, ok := i.heap[f.text]
	if !ok {
		return nil, fmt.Errorf("Error: %s is not a known variable", f.text)
	}
	return obj, nil
}

// find will find things. It is basically a replacement, all purpose object constructor/retriever
func (i *interpreter) find(stype string, id string) (interface{}, error) {
	var obj interface{}
	var err error
	f := i.finders[stype]
	if f == nil {
		return nil, fmt.Errorf("No lookup method found for type %s", stype)
	}
	if obj, err = f(id); err != nil {
		return nil, err
	}
	return obj, nil
}

func (i *interpreter) statementToArgs(mtype reflect.Type, s statement) ([]reflect.Value, error) {
	// No crashing
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	args := make([]reflect.Value, 0)
	// statement should be of the format LPAREN [WORD WORD COMMA] ... RPAREN EOF
	max := len(s)
	if max == 3 {
		// method with no params, return early
		return args, nil
	}
	wordCount := 0
	// TODO this feels like a super hacky way to do this. Improve it?
	for _, currentfrag := range s {
		if isValueToken(currentfrag.token) {
			// hit a comma, reset
			// Get the type of the argument
			tparts := strings.Split(mtype.In(wordCount).String(), ".")
			atype := tparts[len(tparts)-1] // Get whatever is at the final element of the split
			var c Converter
			var ok bool
			if c, ok = i.converters[atype]; !ok {
				return nil, fmt.Errorf("No converter found for type: %s", atype)
			}
			// Convert, error on not found
			iv, err := c(currentfrag.text)
			if err != nil {
				return nil, fmt.Errorf("Error converting %s %s: %s", currentfrag.text, atype, err.Error())
			}
			// Add to the our list to return
			args = append(args, reflect.ValueOf(iv))
			wordCount++ // Could just take len of args over and over but eh
		}
	}
	return args, nil
}

func isValueToken(t Token) bool {
	if STRING <= t && t <= BOOL {
		return true
	}
	return false
}
