# Instructor
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/StabbyCutyou/instructor) 

The world's crappiest joke of a REPL for golang based applications

# What is it?

It's a way to build a sidecar binary that you can configure with database access,
for example, and create a REPL that lets you test loading objects into structs, inspecting
them, and calling basic functions on them.

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

# How do I use the REPL?

Sorry, coming soon. I whipped this up on a whim as a proof of concept, and only
have it on Github at the moment to get feedback. Proper docs are coming, but essentially...

* type: `quit` to exit
* type: `help` to get a list of commands
* type: `find MyStruct MyID`
* type: `call MyStruct MyID SimpleFunc`
* type: `call MyStruct MyID FuncWithParams 50 int true bool 1.0 float64 joejoe string`
* So far, those are the only four param types supported. More coming.

# License

Apache v2 - See LICENSE
