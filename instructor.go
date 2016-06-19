// Package instructor is
package instructor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/tapjoy/adfilteringservice/vendor/github.com/davecgh/go-spew/spew"
)

// Finder is a function type that is used to load an object, serialized into a struct
// from the integrating-applications list of structs
type Finder func(string) (interface{}, error)

// Converter is a function type that is used to convert a string to an associated type
// You'd wrap whatever logic you needed, including something like just JSON Unmarshalling
// to turn a string representation of a value into a concrete instance of it's type.
// Four come out of the box for you: string, bool, int, and float64
type Converter func(string) (interface{}, error)

// Internal types used to be more explicit about the purposes of these maps
type instanceTable map[string]string
type heap map[string]interface{}
type cache map[string]heap
type finders map[string]Finder
type converters map[string]Converter
type arguments []interface{}

// Version is the current semver for this tool
const Version = "0.0.5"

// Instructor is an instance of the object which will allow you to inspect structs
type Instructor struct {
	cache      cache
	finders    finders
	converters converters
	instances  instanceTable
}

type command struct {
	arguments        arguments
	commandName      string
	variableName     string
	variableMethod   string
	variableProperty string
}

// New returns a new Instructor
func New() *Instructor {
	return &Instructor{
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

// RegisterFinder is for registering one of your custom finders to look up your structs
func (i *Instructor) RegisterFinder(name string, f Finder) {
	i.finders[name] = f
}

// RegisterConverter is for registering one of your custom converters to convert cli arguments to typed values
func (i *Instructor) RegisterConverter(name string, c Converter) {
	i.converters[name] = c
}

// This approach is clearly flawed in a number of ways.
// This was done as a proof of concept for the idea of a REPL-sidecar type
// library, to make it so folks could inspect objects and load them from the
// db, network, etc etc. Ideally this should probably be parsing things into units
// of evaluation, or statements, and chaining those statements together via operators
// with some kind of precedence that matches golangs.
// But, you know, it is what it is right now.
func (i *Instructor) buildCommand(input string) (*command, error) {
	// There is 1 special function, find
	// Everything else is done via an instance of a struct
	// find can be alone, or its result can be set to an instance of a struct

	// Walk the string, rune by rune, and parse it into tokens we can handle
	tokenBuffer := bytes.Buffer{}
	foundFunction := false
	cmd := command{}
	var err error
	for _, c := range input {
		switch c {
		case '=':
			// Assignment - the left side, the current buffer, is a variable name
			// We need to pop that off the buffer, and take the right side as what
			// we assign to it

			// Set the variable name with the current contents
			// Trim any whitespace
			cmd.variableName = strings.TrimSpace(tokenBuffer.String())
			// Dump the buffer to prepare for the next set of data
			tokenBuffer.Reset()
		case '(':
			// Begin function params - everything until ) is a string of params for a function call
			// The current tokenBuffer has the function name. Pop that off, save it, and begin
			// collecting the following inputs into the token buffer

			// Trip the foundFunction gate boolean
			foundFunction = true
			f := strings.TrimSpace(tokenBuffer.String())
			// If we're inside the find command, set it as the command name
			if f == "find" {
				cmd.commandName = f
			} else {
				// if we're in any other method, set it as the variable method being invoked
				cmd.variableMethod = f
			}
			tokenBuffer.Reset()
		case ')':
			// End function params - turn the tokenBuffer into a list of arguments
			// This should also be the final character in the input string under the
			// current rules, although every input string will not always have to end with a func call

			// Break the args into parts
			parts := strings.Split(tokenBuffer.String(), ",")

			// Since "find" is a special function of the repl itself, let's fork here to detect
			// if we need to parse the params one way or another
			// TODO robustify this if we add more repl commands
			argParser := i.inputToArgs
			if cmd.commandName == "find" {
				argParser = i.inputToFindArgs
			}

			// Build them up into a properly typed array of interface{}s
			if cmd.arguments, err = argParser(parts); err != nil {
				return nil, err
			}
			tokenBuffer.Reset()
			// Reset the foundFunction gate boolean
			foundFunction = false
		case '.':
			// Dot means we're trying to invoke something on a variable, unless we're in the middle of
			// parsing function params (could be string or float), so treat this case thusly
			if foundFunction {
				// Straight pass through to our buffer
				tokenBuffer.WriteRune(c)
				continue
			}
			// It wasn't inside of a func call, so not args. Treat it like a method/property invocation
			// Set the variable name with the current contents
			// Trim any whitespace
			cmd.variableName = strings.TrimSpace(tokenBuffer.String())
			// Dump the buffer to prepare for the next set of data
			tokenBuffer.Reset()
		default:
			// Straight pass through to our buffer
			tokenBuffer.WriteRune(c)
		}
	}

	// Now that we're done parsing the string, there could be something left in the buffer
	if remaining := tokenBuffer.String(); remaining != "" {
		// If something is remaining...
		// it is a simple call to spew a variable by name
		// or
		// it would be a method or property name after a dot on a variable
		if cmd.variableName == "" {
			// if there wasn't yet a variablename detected, we must have one now
			cmd.variableName = remaining
		} else if cmd.arguments != nil {
			// It was a method
			cmd.variableMethod = remaining
		} else {
			// it was a property
			cmd.variableProperty = remaining
		}
	}

	return &cmd, nil
}

// REPL will enter the read eval print loop, blocking the main thread until it exits
func (i *Instructor) REPL() error {
	// Buffered reader off of STDIN
	reader := bufio.NewReader(os.Stdin)
	input := ""
	stop := false
	var err error
	// Print welcome message
	fmt.Printf("Welcome to Inspector v%s\n", Version)
	fmt.Printf("For a list of commands, type help\n")
	for !stop {
		fmt.Printf("instructor %s >>", Version)
		if input, err = reader.ReadString('\n'); err != nil {
			fmt.Printf("Error reading from STDIN: %s\n", err.Error())
		}
		input = strings.TrimSpace(input)
		switch input {
		case "quit":
			stop = true
		case "help":
			fmt.Println("You can call the following commands:")
			fmt.Println("quit : exits the REPL")
			fmt.Println("help : prints this screen")
			fmt.Println("find : Looks up an object by it's type and ID")
			fmt.Println("\t\tEx: u = find(User,123456789)")
			fmt.Println("You can call methods or invoke Properties on an object. You can provide arguments by giving their type and value, in the order they're defined on the method")
			fmt.Println("\t\tEx: u.Strawmethod(false bool,50 int)")
			fmt.Println("\t\tEc: u.Strawproperty")
		default:
			var cmd *command
			var err error
			if cmd, err = i.buildCommand(input); err != nil {
				fmt.Printf("Error building command: %s\n", err.Error())
			}
			var obj interface{}

			if cmd.commandName == "find" {
				stype := cmd.arguments[0].(string)
				id := cmd.arguments[1].(string)
				//if we're assigning it to a variable or not, we'll do the find regardless
				if obj, err = i.find(stype, id); err != nil {
					fmt.Printf("Error looking up %s %s: %s\n", stype, id, err.Error())
				} else {
					spew.Dump(obj)
				}

				if cmd.variableName != "" {
					// We're storing it by name for later lookups
					var h heap
					var ok bool
					// If the heap for this type doesn't exist, make it
					if h, ok = i.cache[stype]; !ok {
						h = make(heap)
						i.cache[stype] = h
					}
					// Set it on the heap
					h[cmd.variableName] = obj
					// Add a lookup of variable name to type in the instances table
					i.instances[cmd.variableName] = stype
				}
			} else if cmd.variableName != "" {
				// We didn't expressly call one of the built in REPL functions, and there
				// is a variable - so we're either trying to re-spew the variable, or call
				// a method/property on it.
				var obj interface{}
				var ok bool
				var stype string
				// first, get which type it's in
				if stype, ok = i.instances[cmd.variableName]; !ok {
					fmt.Printf("Error: Unknown variable %s\n", cmd.variableName)
					continue
				}
				c := i.cache[stype]
				obj = c[cmd.variableName]
				if cmd.variableMethod == "" && cmd.variableProperty == "" {
					// respew
					spew.Dump(obj)
				} else if cmd.variableProperty != "" {
					// print a property
					i.callProperty(obj, cmd.variableProperty)
				} else {
					// invoke a method
					i.callMethod(obj, cmd.variableMethod, cmd.arguments)
				}
			}
		}
	}
	return nil
}

// find will find things
func (i *Instructor) find(stype string, id string) (interface{}, error) {
	var obj interface{}
	var err error
	f := i.finders[stype]
	if obj, err = f(id); err != nil {
		return nil, err
	}

	return obj, nil
}

func (i *Instructor) callProperty(obj interface{}, propertyName string) error {
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

func (i *Instructor) callMethod(obj interface{}, methodName string, args arguments) error {
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

func (i *Instructor) inputToFindArgs(input []string) (arguments, error) {
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

func (i *Instructor) inputToArgs(input []string) (arguments, error) {
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
