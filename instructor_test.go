package instructor

import (
	"testing"
)

func BenchmarkInputToArgs1Inputs(b *testing.B) {
	i := New()
	input := []string{"true", "bool"}
	for j := 0; j < b.N; j++ {
		i.inputToArgs(input)
	}
}
func BenchmarkInputToArgs2Inputs(b *testing.B) {
	i := New()
	input := []string{"true", "bool", "100", "*int"}
	for j := 0; j < b.N; j++ {
		i.inputToArgs(input)
	}
}
func BenchmarkInputToArgs4Inputs(b *testing.B) {
	i := New()
	input := []string{"true", "bool", "100", "int", "hey", "*string", "500.0", "float64"}
	for j := 0; j < b.N; j++ {
		i.inputToArgs(input)
	}
}
func BenchmarkInputToArgs8Inputs(b *testing.B) {
	i := New()
	input := []string{"true", "bool", "100", "int", "hey", "string", "500.0", "*float64", "true", "bool", "100", "int", "hey", "string", "500.0", "float64"}
	for j := 0; j < b.N; j++ {
		i.inputToArgs(input)
	}
}
func BenchmarkInputToArgs16Inputs(b *testing.B) {
	i := New()
	input := []string{"true", "*bool", "100", "int", "hey", "string", "500.0", "float64", "true", "bool", "100", "int", "hey", "string", "500.0", "float64", "true", "bool", "100", "*uint", "hey", "string", "500.0", "float64", "true", "bool", "100", "uint", "hey", "string", "500.0", "float64"}
	for j := 0; j < b.N; j++ {
		i.inputToArgs(input)
	}
}
