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

// Evaluate is a set of rules dictating how the tokens will be interpreted.
// based on each token, starting with the first, it will walk down a decision
// tree to determine what to do next
// This method is becoming a hard to maintain mess
// Consider: breaking the method up into something that determines what "type"
// of statement it is, and simplify the forking logic?
func (i *interpreter) Evaluate(statement statement) error {
	// First thing, lets just make sure no pesky whitespace is hanging around
	// As a quick cheat, because it's possible, and this is why i'm starting to hate find...
	// We'll check for it first as a special case, and send it down the rabbit hole
	statement = cleanWhitespace(statement)
	firstFragment := statement[0]
	if firstFragment.token == FIND {
		// Pass from the 4th token on (LPAREN <-> RPAREN : EOF))
		// get back arguments for dynamic finder
		stype, id, err := statementToFindArgs(statement[1:])
		if err != nil {
			return err
		}
		obj, err := i.find(stype, id)
		spew.Dump(obj)

	} else if firstFragment.token == WORD {
		// This could be either:
		// xassigning an operation to a variable
		// xcalling a variable (spew results)
		// calling a method on a variable
		// calling a property on a variable
		if statement[1].token == EOF {
			// They called a variable - display it
			// TODO if keywords other than find exist, this needs rethought
			if err := i.printVariable(statement); err != nil {
				return err
			}
		} else if statement[1].token == ASSIGN && len(statement) > 3 {
			// This means we're doing an assignment.
			// Could be from a find call, or a property/method call
			if statement[2].token == FIND {
				//x = find(model, 123)
				// Pass from the 4th token on (LPAREN <-> RPAREN : EOF))
				// get back arguments for dynamic finder
				stype, id, err := statementToFindArgs(statement[3:])
				if err != nil {
					return err
				}
				obj, err := i.find(stype, id)
				if err != nil {
					return err
				}
				// Store it on the heap with the variable name, ie the first fragments text
				if err := i.storeInHeap(firstFragment.text, obj); err != nil {
					return err
				}
				spew.Dump(obj)
			} else if statement[2].token == WORD {
				// something like x = y so far...
				if statement[3].token == PERIOD {
					// we've got something like x = y. - we're going to chain down a series of calls
					// we need to traverse the call stack to get the object to assign to x
					var obj interface{}
					var err error
					chain, args := statementToInvocationChainAndParams(statement)

					if statement[len(statement)-2].token == RPAREN { // If the next to last is a RPAREN, meaning it's a function call
						obj, err = i.callMethodChain(chain[2:], args)
					} else {
						// else, it's a normal property
						obj, err = i.callPropertyChain(chain[2:])
					}
					if err != nil {
						return err
					}
					// Now, assign it to the id on the left
					i.storeInHeap(statement[0].text, obj)

				} else {
					// We've got a simple one, like x = y, where y is a variable or literal
					obj, ok := i.heap[statement[2].text]
					if !ok {
						return fmt.Errorf("Error: unknown variable %s", statement[0].text)
					}
					i.storeInHeap(statement[0].text, obj)
					spew.Dump(obj)
				}
			}
		} else if statement[1].token == PERIOD {
			//x.Method() or x.Property, or a chain of them
			// First, we need to break the statement into pieces:
			// if it's a property call, then we're fine
			// if it's a method call, we need the chain, then the arguments
			chain, args := statementToInvocationChainAndParams(statement)
			if args == nil {
				// It's a property invocation
				// The first word in the statement is the variable
				// the next N are the property chains
				if _, err := i.callPropertyChain(chain); err != nil {
					return err
				}
			} else {
				// It's a method invocation
				// The first word in the statement is the variable
				// the next N are the property chains
				// the args are the things to invoke on the method
				if _, err := i.callMethodChain(chain, args); err != nil {
					return err
				}
			}
		}
	} else {
		return fmt.Errorf("Error: %s is not a valid token at position 0", statement[0])
	}

	return nil
}

func (i *interpreter) callMethodChain(chain statement, args statement) ([]interface{}, error) {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
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

func (i *interpreter) printVariable(s statement) error {
	f := s[0]
	obj, ok := i.heap[f.text]
	if !ok {
		return fmt.Errorf("Error: %s is not a known variable", f.text)
	}

	spew.Dump(obj)
	return nil
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
		if currentfrag.token == WORD {
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
