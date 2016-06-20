package instructor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tapjoy/adfilteringservice/vendor/github.com/davecgh/go-spew/spew"
)

// Current task
// need to test method calls, 1 and N deep.
// After that... code cleanup

// Interpreter is
type Interpreter struct {
	cache      cache
	finders    finders
	converters converters
	instances  instanceTable
}

// NewInterpreter returns a new Instructor
func NewInterpreter() *Interpreter {
	return &Interpreter{
		cache:     make(cache),
		finders:   make(finders),
		instances: make(instanceTable),
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
func (i *Interpreter) Evaluate(statement []fragment) error {
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
		if statement[1].token == EOF && len(statement) == 2 {
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
			} else if statement[2].token == WORD && statement[3].token == PERIOD {
				// we've got something like x = y. - we're going to chain down a series of calls
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
				if err := i.callPropertyChain(chain); err != nil {
					return err
				}
			} else {
				// It's a method invocation
				// The first word in the statement is the variable
				// the next N are the property chains
				// the args are the things to invoke on the method
				if err := i.callMethodChain(chain, args); err != nil {
					return err
				}
			}
		}
	} else {
		return fmt.Errorf("Error: %s is not a valid token at position 0", statement[0])
	}

	return nil
}

func (i *Interpreter) callMethodChain(chain []fragment, args []fragment) error {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	var err error
	inputArgs, err := i.statementToArgs(args)
	if err != nil {
		return err
	}
	max := len(chain)
	// Get the object to call the method on
	// if max-1 is the last element, and that is the funcion,
	var obj interface{}
	if max > 2 {
		// if max > 2, then the chain needs to be evaluated to get the object
		// to call the method on. Otherwise, the 2nd fragment is the method, and
		// it's called directly
		obj, err = i.crawlPropertyChain(chain[:max-1])
		if err != nil {
			return err
		}
	} else {
		stype, ok := i.instances[chain[0].text]
		if !ok {
			return fmt.Errorf("Error: Unknown variable %s", chain[0].text)
		}
		heap, ok := i.cache[stype]
		if !ok {
			return fmt.Errorf("Error: Unknown variable %s", chain[0].text)
		}
		obj, ok = heap[chain[0].text]
		if !ok {
			return fmt.Errorf("Error: Unknown variable %s", chain[0].text)
		}
	}

	// Get the reflect value and look up the method
	v := reflect.ValueOf(obj)
	// Don't do this for methods.. but perhaps we need a ptr/nonptr fallback?
	//if v.Kind() == reflect.Ptr {
	//	v = v.Elem()
	//}
	m := v.MethodByName(chain[max-1].text)
	// Make a method of params to pass into the Method
	vArgs := make([]reflect.Value, len(inputArgs))
	for i, arg := range inputArgs {
		vArgs[i] = reflect.ValueOf(arg)
	}

	// Call the Method with the value args
	r := m.Call(vArgs)
	// Print all the results
	spew.Dump(r)
	return nil
}

func (i *Interpreter) crawlPropertyChain(statement []fragment) (interface{}, error) {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	stype, ok := i.instances[statement[0].text]
	if !ok {
		return nil, fmt.Errorf("Error: Unknown variable %s", statement[0].text)
	}
	h, ok := i.cache[stype]
	if !ok {
		return nil, fmt.Errorf("Error: Unknown variable %s", statement[0].text)
	}
	obj, ok := h[statement[0].text]
	if !ok {
		return nil, fmt.Errorf("Error: Unknown variable %s", statement[0].text)
	}
	currentVal := reflect.ValueOf(obj)

	for _, f := range statement[1:] {
		if f.token != EOF {
			// Deref if we're dealing with a pointer
			if currentVal.Kind() == reflect.Ptr {
				currentVal = currentVal.Elem()
			}
			p := currentVal.FieldByName(f.text)
			currentVal = p
		}
	}

	return currentVal.Interface(), nil
}

func (i *Interpreter) callPropertyChain(statement []fragment) error {
	obj, err := i.crawlPropertyChain(statement)
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", obj)
	return nil
}

func statementToInvocationChainAndParams(statement []fragment) ([]fragment, []fragment) {
	chain := make([]fragment, 0)
	args := make([]fragment, 0)
	hitParen := false
	for _, f := range statement {
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

func statementToFindArgs(statement []fragment) (string, string, error) {
	max := len(statement)
	if max == 6 && // Find statement should only have 7 fragements
		statement[0].token == LPAREN &&
		statement[max-1].token == EOF &&
		statement[max-2].token == RPAREN {
		// Valid set of args so far
		return statement[1].text, statement[3].text, nil
	}
	// TODO re-assemble statement for error message
	return "", "", fmt.Errorf("Error: Invalid set of arguments for Find")
}

func cleanWhitespace(frags []fragment) []fragment {
	results := make([]fragment, 0)
	for _, f := range frags {
		if f.token != WS {
			results = append(results, f)
		}
	}
	return results
}

// RegisterFinder is for registering one of your custom finders to look up your structs
func (i *Interpreter) RegisterFinder(name string, f Finder) {
	i.finders[name] = f
}

// RegisterConverter is for registering one of your custom converters to convert cli arguments to typed values
func (i *Interpreter) RegisterConverter(name string, c Converter) {
	i.converters[name] = c
}

func (i *Interpreter) storeInHeap(id string, obj interface{}) error {
	// Get the type as a string
	t := reflect.TypeOf(obj)
	stype := strings.Split(t.String(), ".")[1]
	// Store record in i.instances
	i.instances[id] = stype
	// Get it's heap from cache
	h, ok := i.cache[stype]
	if !ok {
		// if none, make it
		h = make(heap)
		i.cache[stype] = h
	}
	// store obj by id.
	h[id] = obj
	// TODO is any of this necessary? Couldn't I just store instances in a hash
	// without the indirection of the current type based heap cache lookup? Seems
	// overdone. The way it is, I could eventually do something type-specific with it
	// I suppose... but will / can i?

	return nil
}

func (i *Interpreter) printVariable(statement []fragment) error {
	f := statement[0]
	stype, ok := i.instances[f.text]
	if !ok {
		return fmt.Errorf("Error: %s is not a known variable", f.text)
	}
	h, ok := i.cache[stype]
	if !ok {
		return fmt.Errorf("Error: %s is not a known variable", f.text)
	}
	obj, ok := h[f.text]
	if !ok {
		return fmt.Errorf("Error: %s is not a known variable", f.text)
	}
	spew.Dump(obj)
	return nil
}

// find will find things. It is basically a replacement, all purpose object constructor/retriever
func (i *Interpreter) find(stype string, id string) (interface{}, error) {
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

func (i *Interpreter) callMethod(obj interface{}, methodName string, args arguments) error {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	// Get the reflect value and look up the method
	v := reflect.ValueOf(obj)
	m := v.MethodByName(methodName)
	// Make a method of params to pass into the Method
	vArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		vArgs[i] = reflect.ValueOf(arg)
	}
	// Call the Method with the value args
	r := m.Call(vArgs)
	// Print all the results
	for _, rv := range r {
		fmt.Printf("%+v\n", rv)
	}
	return nil
}

func (i *Interpreter) statementToArgs(statement []fragment) (arguments, error) {
	// No crashing
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	args := make(arguments, 0)
	// statement should be of the format LPAREN [WORD WORD COMMA] ... RPAREN EOF
	max := len(statement)
	if max == 3 {
		// method with no params, return early
		return args, nil
	}
	j := 1
	currentfrag := statement[j] // skip the first paren
	currentval := ""
	currenttype := ""
	// TODO this feels like a super hacky way to do this. Improve it?
	for currentfrag.token != EOF {
		if currentfrag.token == WORD {
			if currentval == "" {
				currentval = currentfrag.text
			} else if currenttype == "" {
				currenttype = currentfrag.text
			}
		} else if currentfrag.token == COMMA || currentfrag.token == RPAREN {
			// hit a comma, reset
			var c Converter
			var ok bool
			if c, ok = i.converters[currenttype]; !ok {
				return nil, fmt.Errorf("No converter found for type: %s", currenttype)
			}
			// Convert, error on not found
			iv, err := c(currentval)
			if err != nil {
				return nil, fmt.Errorf("Error converting %s %s: %s", currentval, currenttype, err.Error())
			}
			// Add to the our list to return
			args = append(args, iv)
			currentval = ""
			currenttype = ""
		}

		j++
		currentfrag = statement[j]
	}
	return args, nil
}
