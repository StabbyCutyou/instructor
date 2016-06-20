package instructor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tapjoy/adfilteringservice/vendor/github.com/davecgh/go-spew/spew"
)

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

	} else if firstFragment.token == WORD {
		// This could be either:
		// assigning an operation to a variable
		// calling a variable (spew results)
		// calling a method on a variable
		// calling a property on a variable
		if len(statement) == 1 {
			// They called a variable - display it
			// TODO if keywords other than find exist, this needs rethought
			i.printVariable(statement)
		}
		if statement[1].token == ASSIGN && len(statement) > 3 {
			//x = find(model, 123)
			if statement[2].token == FIND {
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
				if err := i.storeInHeap(id, obj); err != nil {
					return err
				}
				spew.Dump(obj)
			}
		} else if statement[1].token == PERIOD {
			//x.Method() or x.Property
		}
	} else {
		return fmt.Errorf("Error: %s is not a valid token at position 0", statement[0])
	}

	return nil
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
	stype := t.String()
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

// find will find things
func (i *Interpreter) find(stype string, id string) (interface{}, error) {
	var obj interface{}
	var err error
	f := i.finders[stype]
	if obj, err = f(id); err != nil {
		return nil, err
	}

	return obj, nil
}

func (i *Interpreter) callProperty(obj interface{}, propertyName string) error {
	// No crashing!
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	v := reflect.ValueOf(obj).Elem()
	p := v.FieldByName(propertyName)
	r := p.Interface()
	// Print all the results
	fmt.Printf("%+v\n", r)
	return nil
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

func (i *Interpreter) inputToFindArgs(input []string) (arguments, error) {
	// No crashing
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	// This will just handle the 2 arguments find expects
	args := make(arguments, 2)
	if len(input) != 2 {
		return nil, fmt.Errorf("Incorrect number of arguments passed to find: Expected 2, got %d", len(input))
	}
	args[0] = input[0]
	args[1] = input[1]
	return args, nil
}

func (i *Interpreter) inputToArgs(input []string) (arguments, error) {
	// No crashing
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Recovering from panic: %s\n", err)
		}
	}()
	args := make(arguments, 0)
	if len(input) == 1 && input[0] == "" {
		// Edge case for functions parsed with no params
		return args, nil
	}
	// For every input
	for _, arg := range input {
		parts := strings.Split(arg, " ")
		val := strings.TrimSpace(parts[0])
		t := strings.TrimSpace(parts[1])
		var c Converter
		var ok bool
		// Lookup our converter, error on not found
		if c, ok = i.converters[t]; !ok {
			return nil, fmt.Errorf("No converter found for type: %s", t)
		}
		// Convert, error on not found
		iv, err := c(val)
		if err != nil {
			return nil, fmt.Errorf("Error converting %s %s: %s", val, t, err.Error())
		}
		// Add to the our list to return
		args = append(args, iv)
	}
	return args, nil
}
