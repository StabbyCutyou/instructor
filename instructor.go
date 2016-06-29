// Package instructor is
package instructor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
type heap map[string]interface{}
type finders map[string]Finder
type converters map[string]Converter
type fragment struct {
	token Token
	text  string
}
type statement []fragment

// Version is the current semver for this tool
const Version = "0.1.9"

// Instructor is an instance of the object which will allow you to inspect structs
type Instructor struct {
	interpreter *interpreter
}

// New returns a new Instructor
func New() *Instructor {
	return &Instructor{
		interpreter: newInterpreter(),
	}
}

// RegisterFinder is for registering one of your custom finders to look up your structs
func (i *Instructor) RegisterFinder(name string, f Finder) {
	i.interpreter.RegisterFinder(name, f)
}

// RegisterConverter is for registering one of your custom converters to convert cli arguments to typed values
func (i *Instructor) RegisterConverter(name string, c Converter) {
	i.interpreter.RegisterConverter(name, c)
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
			fmt.Println("\t\tEx: u = find(User,\"123456789\")")
			fmt.Println("You can call methods or invoke Properties on an object. You can provide arguments by giving their type and value, in the order they're defined on the method")
			fmt.Println("\t\tEx: u.Strawmethod(false ,50)")
			fmt.Println("\t\tEc: u.Strawproperty")
		default:
			l := newLexer(strings.NewReader(input))
			var f fragment
			s := make(statement, 0)
			for f.token != EOF {
				f = l.scan()
				s = append(s, f)
			}
			// TODO just lass the lexer straight into evaluate
			i.interpreter.Evaluate(s)
		}
	}
	return nil
}
