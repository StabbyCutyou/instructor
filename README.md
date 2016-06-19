# Instructor
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/StabbyCutyou/instructor)

The world's crappiest joke of a "REPL" for golang based applications

# What is it?

It's a way to build a sidecar binary that you can configure with database access,
for example, and create a "REPL" that lets you interact with data, inspecting
structs, and calling basic functions / properties on them.

# Wait, this isn't actually a REPL

Nah, it isn't.

It's more of a sidecar utility you can bolt on to an existing application, and use
to do some basic interactions with your applications data in a REPL-like environment.

# Yea, but how fast is it? I'm *very* particular about code that doesn't perform well

Well, that's a real shame. Cus this isn't meant to be fast. It certainly isn't "slow", but
the code right now is a proof of concept and is likely to change and be unstable. It's also
not my current focus to be super performant, but rather to refine the approach and add features.

I'm not against performance improvements, they just aren't my focus at the moment.

# Wow, looking at the code, it is clear you have no idea how to write an interpreter

I sure don't! Part of the fun was trying to make it work.

# Why didn't you...

See previous answer

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
```

# How do I use it?

Once you have it integrated as a sidecar via a tool or cmd binary, you would simply
invoke the binary. This will drop you into an interactive prompt, where you can perform
things like the following:

* type: `quit` to exit
* type: `help` to get a list of commands
* type: `o = find(MyStruct, MyID)`
  * To use the find helper, you'll need to use RegisterFinder, as demonstrated in the sample code above
* type: `o.Property`
* type: `o.SimpleFunc()`
* type: `o.ComplexFunc(50 int, true bool)`
* So far, those are the following param types supported:
 * int
 * *int
 * uint
 * *uint
 * float64
 * *float64
 * string
 * *string
 * bool
 * *bool

# License

Apache v2 - See LICENSE
