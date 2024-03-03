package tffunc

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"

	"github.com/apparentlymart/go-tf-func-provider/internal/tfplugin6"
)

// Provider represents a functions-only provider for Terraform, or other
// software that can act as a client for Terraform's plugin protocol.
type Provider struct {
	funcs   map[string]providerFunction
	schemas map[string]*tfplugin6.Function
	metas   []*tfplugin6.GetMetadata_FunctionMetadata
}

type providerFunction struct {
	impl         function.Function
	paramTypes   []cty.Type
	varParamType cty.Type
}

// NewProvider constructs a new [Provider] that initially supports no functions
// at all.
//
// Use [Provider.AddFunction] calls to register one or more functions before
// calling [Provider.Serve] to start the plugin server.
func NewProvider() *Provider {
	return &Provider{
		funcs:   make(map[string]providerFunction),
		schemas: make(map[string]*tfplugin6.Function),
	}
}

// AddFunction adds a new function to the provider with the given name and
// specification.
//
// Function names must be unique. If a caller tries to add a function whose
// name was previously used for another call then this function will panic.
//
// The caller must not access or mutate anything reachable from the spec
// pointer after calling this function.
//
// When calling functions from plugins Terraform handles "marks" such as
// sensitivity automatically before making any provider calls, and so it
// isn't valid to define a function with any parameter having AllowMarked
// set. Trying to define such a function will cause a panic.
//
// Terraform requires that all provider-contributed functions act as "pure"
// functions, meaning that their results are decided entirely based on the
// arguments. For example, it's not valid to write a function that returns
// a random number unless the seed for random number generation is one of the
// arguments to the function.
func (p *Provider) AddFunction(name string, spec *function.Spec) {
	if _, exists := p.funcs[name]; exists {
		panic(fmt.Sprintf("function %q was already defined", name))
	}
	schema := functionSchema(spec) // will panic if the spec is invalid
	fn := buildFunction(spec)
	p.funcs[name] = fn
	p.schemas[name] = schema
	p.metas = append(p.metas, &tfplugin6.GetMetadata_FunctionMetadata{
		Name: name,
	})
}

// Serve attempts to start a plugin server after negotiating with the parent
// process that is presumably the plugin client.
//
// If successful, this function never returns.
//
// Returns errors if the server cannot start for any dynamic reason, such as
// if the protocol negotiation fails.
func (p *Provider) Serve(ctx context.Context) error {
	return rpcplugin.Serve(ctx, &rpcplugin.ServerConfig{
		Handshake: rpcplugin.HandshakeConfig{
			CookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
			CookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
		},
		ProtoVersions: map[int]rpcplugin.ServerVersion{
			6: rpcplugin.ServerVersionFunc(func(s *grpc.Server) error {
				tfplugin6.RegisterProviderServer(s, &pluginServer6{
					p: p,
				})
				return nil
			}),
		},
	})
}

// CallStub returns a Go function pointer that wraps the function of the given
// name, or panics if there is no such function registered.
//
// This is here primarily for use in unit tests, which can use this function
// instead of calling [Provider.Serve] to test the behavior of individual
// functions in isolation.
func (p *Provider) CallStub(funcName string) func(...cty.Value) (cty.Value, error) {
	f, exists := p.funcs[funcName]
	if !exists {
		panic(fmt.Sprintf("call stub request for undefined function %q", funcName))
	}
	return func(args ...cty.Value) (cty.Value, error) {
		return f.impl.Call(args)
	}
}

func buildFunction(spec *function.Spec) providerFunction {
	var paramTypes []cty.Type
	var varParamType cty.Type
	if len(spec.Params) != 0 {
		paramTypes = make([]cty.Type, len(spec.Params))
		for i, p := range spec.Params {
			paramTypes[i] = p.Type
		}
	}
	if spec.VarParam != nil {
		varParamType = spec.VarParam.Type
	}

	impl := function.New(spec)
	return providerFunction{
		impl:         impl,
		paramTypes:   paramTypes,
		varParamType: varParamType,
	}
}

func functionSchema(spec *function.Spec) *tfplugin6.Function {
	ret := &tfplugin6.Function{
		Description:     spec.Description,
		DescriptionKind: tfplugin6.StringKind_PLAIN,
	}
	for _, p := range spec.Params {
		pSchema := parameterSchema(&p)
		ret.Parameters = append(ret.Parameters, pSchema)
	}
	if spec.VarParam != nil {
		pSchema := parameterSchema(spec.VarParam)
		ret.VariadicParameter = pSchema
	}
	ret.Return = &tfplugin6.Function_Return{
		// cty functions are allowed to decide their return types dynamically
		// based on given argument values, so we'll just always report
		// cty.DynamicPseudoType here and then serialize the real type as
		// part of the function call result.
		Type: dynamicPseudoTypeRaw,
	}
	return ret
}

func parameterSchema(spec *function.Parameter) *tfplugin6.Function_Parameter {
	if spec.AllowMarked {
		panic(fmt.Sprintf("parameter %q sets AllowMarked, which is forbidden", spec.Name))
	}
	ret := &tfplugin6.Function_Parameter{
		Name:               spec.Name,
		AllowNullValue:     spec.AllowNull,
		AllowUnknownValues: spec.AllowUnknown,
		Description:        spec.Description,
		DescriptionKind:    tfplugin6.StringKind_PLAIN,
	}

	tyRaw, err := spec.Type.MarshalJSON()
	if err != nil {
		panic(fmt.Sprintf("parameter %q has unsupported type %#v: %s", spec.Name, spec.Type, err))
	}
	ret.Type = tyRaw

	return ret
}

var dynamicPseudoTypeRaw []byte

func init() {
	var err error
	dynamicPseudoTypeRaw, err = cty.DynamicPseudoType.MarshalJSON()
	if err != nil {
		// if we get here then cty is buggy
		panic("can't serialize cty.DynamicPseudoType")
	}
}
