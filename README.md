Instructor
======================

The world's crappiest joke of a REPL for golang based applications

What is it?
======================

It's a way to build a sidecar binary that you can configure with database access,
for example, and create a REPL that lets you test loading objects into structs, inspecting
them, and calling basic functions on them.

How do I use it?
======================
Sorry, coming soon. I whipped this up on a whim as a proof of concept, and only
have it on Github at the moment to get feedback. Proper docs are coming, but essentially...

* Define a lookup method with this signature `func(string)(interface{},error)` for each struct you want to look up
* Create an Instructor and call REPL
* type: `find MyStruct MyID`
* type: `call MyStruct MyID SimpleFunc`
* type: `call MyStruct MyID FuncWithParams 50 int true bool 1.0 float64`
* So far, those are the only three param types supported. More coming.
