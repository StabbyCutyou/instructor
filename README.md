# Instructor
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/StabbyCutyou/instructor)

It's like `rails c` for go applications, except not nearly as good

# What is it?

It's a way to build a sidecar binary that you can configure with access to objects
in your applications or services. This includes loading data from a database or another
remote location, and invoking methods and properties on those objects interactively.

If what you want to is to glue together a little bootstrap code, so you can hop into a REPL-like environment and inspect data in your go app via structs, methods, and properties, Instructor might be for you.

# What isn't it?

It's not a full fledged go environment - there are many things you cannot do. Even
basic things, like declaring new instances in a "normal" way, or basic addition,
or even declaring pointers!

While all that sounds limiting, and it is, there are ways to use Instructor and
account for those issues as well, albeit with some upfront work via configuration.

Additionally, as I encounter more use cases, I focus on them for development. So
the list of features it doesn't support is essentially the set of features that
weren't immediately pressing while developing it

# Wait, so this isn't actually a REPL?

Nah, it isn't.

It's more of a sidecar utility you can bolt on to an existing application, and use
to do some basic interactions with your applications data in an interactive environment.

Kind of like a REPL. But I actually called Instructor that, i'd be rightfully torn to pieces.

# Yea, but how fast is it? I'm *very* particular about code that doesn't perform well

Well, that's a real shame. Cus this isn't meant to be fast. It certainly isn't "slow", but
the code is likely to change wildly at any time, so until I'm happy with an approach, my
intention is to focus more on features, and move to any performance issues once the internals
are stable.

Instructor makes heavy use of `interface{}` and the `reflect` package, so there will
generally be some unavoidable performance hit at scale for the things it does.

But ideally, the primary use case of someone looking to use Instructor is for convenience,
not performance.

# Wow, looking at the code, it is clear you have no idea how to write an interpreter

I sure don't! Part of the fun was trying to make it work.

# Why not use yacc / bison / etc?

It's more fun to learn this way

# Any roadmap?
* Lots of code cleanup and improvements
* Find a better solution than my hackneyed "find" method for seeding objects into the environment. Ideally, you could just "make"/"do" whatever you want, but that's pie in the sky stuff.
* Variable substitution when calling methods
* Array/Slice indexing
* Setting Properties on objects

# How do I integrate it into my app?

Currently, the recommended way to integrate into an existing app is create a sub-binary
to act as your CLI / REPL. Common patterns for this might include:

`my_app/cmd/repl/repl.go`
`my_app/tools/repl/repl.go`

Inside of `repl.go`, you might have some code like the following

```go
package main

import (
	"github.com/StabbyCutyou/instructor"
	"github.com/you/yourapp/models"
	"github.com/you/youapp/services"
)

var structFinder services.MyStructService

type Flooper struct {
  floops int
}

func main() {
  // Configure access to your database
  // ...
  // Do any additional config so you can boot into a facsimile of your app
  // ex: load any needed files, configure any needed global state, etc etc
  structFinder = services.NewMyStructService()
  // Create an instructor
  i := instructor.New()
  // Register a lookup method for MyStructs
  i.RegisterFinder("MyStruct", findMyStruct)
  i.RegisterConverter("Flooper", convertFloopers)

  // This will block until done
  if err := i.REPL(); err != nil {
    log.Fatal(err)
  }
}

func findMyStruct(string id)(interface{}, error) {
  i, err := strconv.Atoi(id)
  if err != nil {
    return structFinder.Find(id)
  }
  return nil, err
}

func convertFloopers(string s) (interface{}) {
  b := []byte(s)
  f := Flooper{}
  if err := json.Unmarshal(b, &f); err != nil {
    fmt.Println(s)
    fmt.Println(err)
    return f, err
  }
  return f, nil
}
```

# How do I use it?

Once you have it integrated as a sidecar via a tool or cmd binary, you would simply
invoke the binary. This will drop you into an interactive prompt, where you can perform
things like the following:

* type: `quit` to exit
* type: `help` to get a list of commands
* type: `o = find(MyStruct, "MyID")`
  * To use the find helper, you'll need to use RegisterFinder, as demonstrated in the sample code above
* type: `o.Property`
* type: `o.SimpleFunc()`
* type: `o.ComplexFunc(50, true)`
* So far, those are the following param types supported:
 * int
 * uint
 * float64
 * string
 * rune
 * bool
 * Pointer types of the above are technically supported, but you can't make literals of them yet - this is coming shortly
 * Additionally, you can define a "Custom Converter" for any type you want, so long as you can find a way to marshall it from string.
  * In the example above, you could pass a "Flooper" to a method by using the following string
  * `{"floops": 5}`
  * See lexer_test.go for an example

# License
Apache v2 - See LICENSE
