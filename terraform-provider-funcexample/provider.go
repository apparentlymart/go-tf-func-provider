package main

import (
	"strings"

	"github.com/apparentlymart/go-tf-func-provider/tffunc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func newProvider() *tffunc.Provider {
	p := tffunc.NewProvider()

	p.AddFunction("upper", &function.Spec{
		Description: "Converts a given string to uppercase.",
		Params: []function.Parameter{
			{
				Name:        "str",
				Description: "The string to convert.",
				Type:        cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// The tffunc library guarantees that args has at least enough
			// elements to cover the declared non-variadic parameters and
			// that they each conform to the given parameter specification.
			s := args[0].AsString()
			return cty.StringVal(strings.ToUpper(s)), nil
		},
		RefineResult: func(rb *cty.RefinementBuilder) *cty.RefinementBuilder {
			// This function never returns a null value
			return rb.NotNull()
		},
	})
	p.AddFunction("lower", &function.Spec{
		Description: "Converts a given string to lowercase.",
		Params: []function.Parameter{
			{
				Name:        "str",
				Description: "The string to convert.",
				Type:        cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// The tffunc library guarantees that args has at least enough
			// elements to cover the declared non-variadic parameters and
			// that they each conform to the given parameter specification.
			s := args[0].AsString()
			return cty.StringVal(strings.ToLower(s)), nil
		},
		RefineResult: func(rb *cty.RefinementBuilder) *cty.RefinementBuilder {
			// This function never returns a null value
			return rb.NotNull()
		},
	})

	return p
}
