// Package instructor is
package instructor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
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

type cache map[string]map[string]interface{}
type finders map[string]Finder
type converters map[string]Converter

// Version is the current semver for this tool
const Version = "0.0.2"

// Instructor is
type Instructor struct {
	cache      cache
	finders    finders
	converters converters
}

// New returns a new Instructor
func New() *Instructor {
	return &Instructor{
		cache:      make(cache),
		finders:    make(finders),
		converters: map[string]Converter{"bool": stringToBool, "int": stringToInt, "float64": stringToFloat64, "string": stringToString},
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
			fmt.Println("\t\tEx: find User 123456789")
			fmt.Println("call : Calls a method on an object that has already been found in the cache. You can provide arguments by giving their type and value, in the order they're defined on the method")
			fmt.Println("\t\tEx: call User 123456789 Strawmethod false bool, 50 int")
		default:
			parts := strings.Split(input, " ")
			var obj interface{}
			var args []interface{}
			var err error
			switch parts[0] {
			case "find":
				if obj, err = i.find(parts[1], parts[2]); err != nil {
					fmt.Println(parts)
					fmt.Println(err)
					fmt.Printf("Error looking up %s %s: %s\n", parts[1], parts[2], err.Error())
				} else {
					fmt.Println("lllll")
					spew.Dump(obj)
				}
			case "call":
				args, err = i.inputToArgs(parts[4:])
				if err != nil {
					fmt.Printf("Error calling %s %s %s: %s\n", parts[1], parts[2], parts[3], err.Error())
				}
				if err = i.call(parts[1], parts[2], parts[3], args); err != nil {
					fmt.Printf("Error calling %s %s %s: %s\n", parts[1], parts[2], parts[3], err.Error())
				}
			default:
				fmt.Printf("Unknown command %s\n", parts[0])
			}
		}
	}
	return nil
}

// find will find things
func (i *Instructor) find(stype string, id string) (interface{}, error) {
	fmt.Printf("FINDING %s %s\n", stype, id)
	var sc map[string]interface{}
	var ok bool
	// Look up the cache for this type
	if sc, ok = i.cache[stype]; !ok {
		// If not found, make one
		sc = make(map[string]interface{})
		i.cache[stype] = sc
	}
	var obj interface{}
	var err error
	// Look up this object in the cache
	if obj, ok = sc[id]; !ok {
		// If not found, find it
		f := i.finders[stype]
		if obj, err = f(id); err != nil {
			return nil, err
		}
		sc[id] = obj
	}
	// Return the object
	return obj, nil
}

func (i *Instructor) call(stype string, id string, methodName string, args []interface{}) error {
	var obj interface{}
	var err error
	// Get the object, either from cache or from the underlying call to look it up
	if obj, err = i.find(stype, id); err != nil {
		return err
	}
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

func (i *Instructor) inputToArgs(input []string) ([]interface{}, error) {
	// Check the length, if it's lopsided it can't be right
	if len(input)%2 != 0 {
		return nil, errors.New("Input arguments incorrectly formatted")
	}
	args := make([]interface{}, 0)
	// For every pair of inputs
	for j := 0; j < len(input); j += 2 {
		val := input[j]
		t := input[j+1]
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

// All converter functions down here
func stringToBool(s string) (interface{}, error) {
	return strconv.ParseBool(s)
}

func stringToInt(s string) (interface{}, error) {
	return strconv.Atoi(s)
}

func stringToFloat64(s string) (interface{}, error) {
	return strconv.ParseFloat(s, 64)
}

func stringToString(s string) (interface{}, error) {
	return s, nil
}
