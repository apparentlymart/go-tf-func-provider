package tffunc

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/apparentlymart/go-tf-func-provider/internal/tfplugin6"
)

type pluginServer6 struct {
	p *Provider
}

var _ tfplugin6.ProviderServer = (*pluginServer6)(nil)

// ApplyResourceChange implements tfplugin6.ProviderServer.
func (p *pluginServer6) ApplyResourceChange(context.Context, *tfplugin6.ApplyResourceChange_Request) (*tfplugin6.ApplyResourceChange_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// CallFunction implements tfplugin6.ProviderServer.
func (p *pluginServer6) CallFunction(ctx context.Context, req *tfplugin6.CallFunction_Request) (*tfplugin6.CallFunction_Response, error) {
	name := req.Name
	rawArgs := req.Arguments

	fn, exists := p.p.funcs[name]
	// The following errors should all not actually happen if the plugin client
	// is implemented correctly, since it should ensure that the arguments
	// match the schema. We're checking these just to be robust, but not worrying
	// too much about returning good error messages because nobody should see
	// these messages anyway.
	if !exists {
		return &tfplugin6.CallFunction_Response{
			Error: &tfplugin6.FunctionError{
				Text: fmt.Sprintf("this provider does not offer a function named %q", name),
			},
		}, nil
	}
	if fn.varParamType == cty.NilType {
		if len(rawArgs) != len(fn.paramTypes) {
			return &tfplugin6.CallFunction_Response{
				Error: &tfplugin6.FunctionError{
					Text: fmt.Sprintf("argument count must be %d", len(fn.paramTypes)),
				},
			}, nil
		}
	} else {
		if len(rawArgs) < len(fn.paramTypes) {
			return &tfplugin6.CallFunction_Response{
				Error: &tfplugin6.FunctionError{
					Text: fmt.Sprintf("argument count must be at least %d", len(fn.paramTypes)),
				},
			}, nil
		}
	}

	args := make([]cty.Value, len(rawArgs))
	for i, raw := range rawArgs {
		var wantTy cty.Type
		if i < len(fn.paramTypes) {
			wantTy = fn.paramTypes[i]
		} else {
			wantTy = fn.varParamType
		}

		var v cty.Value
		var err error
		// Clients are allowed to encode each argument using either JSON or MessagePack
		switch {
		case len(raw.Json) != 0:
			v, err = json.Unmarshal(raw.Json, wantTy)
		case len(raw.Msgpack) != 0:
			v, err = msgpack.Unmarshal(raw.Msgpack, wantTy)
		default:
			// Should not get here, because if a later version of the protocol
			// introduces a new serialization format then it should be
			// negotiated as a new server capability, which this plugin would
			// then not advertise.
			argIdx := int64(i)
			return &tfplugin6.CallFunction_Response{
				Error: &tfplugin6.FunctionError{
					Text:             "plugin client is using unsupported argument encoding format",
					FunctionArgument: &argIdx,
				},
			}, nil
		}
		if err != nil {
			argIdx := int64(i)
			return &tfplugin6.CallFunction_Response{
				Error: &tfplugin6.FunctionError{
					Text:             fmt.Sprintf("invalid encoding for argument: %s", err),
					FunctionArgument: &argIdx,
				},
			}, nil
		}

		args[i] = v
	}

	result, err := fn.impl.Call(args)
	switch err := err.(type) {
	case nil:
		// Success!
	case function.ArgError:
		argIdx := int64(err.Index)
		return &tfplugin6.CallFunction_Response{
			Error: &tfplugin6.FunctionError{
				Text:             err.Error(),
				FunctionArgument: &argIdx,
			},
		}, nil
	default:
		return &tfplugin6.CallFunction_Response{
			Error: &tfplugin6.FunctionError{
				Text: err.Error(),
			},
		}, nil
	}

	resultRaw, err := msgpack.Marshal(result, cty.DynamicPseudoType)
	if err != nil {
		return &tfplugin6.CallFunction_Response{
			Error: &tfplugin6.FunctionError{
				Text: fmt.Sprintf("failed to encode result: %s", err),
			},
		}, nil
	}

	return &tfplugin6.CallFunction_Response{
		Result: &tfplugin6.DynamicValue{
			Msgpack: resultRaw,
		},
	}, nil
}

// ConfigureProvider implements tfplugin6.ProviderServer.
func (p *pluginServer6) ConfigureProvider(context.Context, *tfplugin6.ConfigureProvider_Request) (*tfplugin6.ConfigureProvider_Response, error) {
	return &tfplugin6.ConfigureProvider_Response{}, nil
}

// GetFunctions implements tfplugin6.ProviderServer.
func (p *pluginServer6) GetFunctions(context.Context, *tfplugin6.GetFunctions_Request) (*tfplugin6.GetFunctions_Response, error) {
	return &tfplugin6.GetFunctions_Response{
		Functions: p.p.schemas,
	}, nil
}

// GetMetadata implements tfplugin6.ProviderServer.
func (p *pluginServer6) GetMetadata(context.Context, *tfplugin6.GetMetadata_Request) (*tfplugin6.GetMetadata_Response, error) {
	return &tfplugin6.GetMetadata_Response{
		ServerCapabilities: serverCapabilities6,
		Functions:          p.p.metas,
	}, nil
}

// GetProviderSchema implements tfplugin6.ProviderServer.
func (p *pluginServer6) GetProviderSchema(context.Context, *tfplugin6.GetProviderSchema_Request) (*tfplugin6.GetProviderSchema_Response, error) {
	return &tfplugin6.GetProviderSchema_Response{
		Provider: &tfplugin6.Schema{
			Block: &tfplugin6.Schema_Block{},
		},
		ServerCapabilities: serverCapabilities6,
		Functions:          p.p.schemas,
	}, nil
}

// ImportResourceState implements tfplugin6.ProviderServer.
func (p *pluginServer6) ImportResourceState(context.Context, *tfplugin6.ImportResourceState_Request) (*tfplugin6.ImportResourceState_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// MoveResourceState implements tfplugin6.ProviderServer.
func (p *pluginServer6) MoveResourceState(context.Context, *tfplugin6.MoveResourceState_Request) (*tfplugin6.MoveResourceState_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// PlanResourceChange implements tfplugin6.ProviderServer.
func (p *pluginServer6) PlanResourceChange(context.Context, *tfplugin6.PlanResourceChange_Request) (*tfplugin6.PlanResourceChange_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// ReadDataSource implements tfplugin6.ProviderServer.
func (p *pluginServer6) ReadDataSource(context.Context, *tfplugin6.ReadDataSource_Request) (*tfplugin6.ReadDataSource_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// ReadResource implements tfplugin6.ProviderServer.
func (p *pluginServer6) ReadResource(context.Context, *tfplugin6.ReadResource_Request) (*tfplugin6.ReadResource_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// StopProvider implements tfplugin6.ProviderServer.
func (p *pluginServer6) StopProvider(context.Context, *tfplugin6.StopProvider_Request) (*tfplugin6.StopProvider_Response, error) {
	return &tfplugin6.StopProvider_Response{}, nil
}

// UpgradeResourceState implements tfplugin6.ProviderServer.
func (p *pluginServer6) UpgradeResourceState(context.Context, *tfplugin6.UpgradeResourceState_Request) (*tfplugin6.UpgradeResourceState_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// ValidateDataResourceConfig implements tfplugin6.ProviderServer.
func (p *pluginServer6) ValidateDataResourceConfig(context.Context, *tfplugin6.ValidateDataResourceConfig_Request) (*tfplugin6.ValidateDataResourceConfig_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

// ValidateProviderConfig implements tfplugin6.ProviderServer.
func (p *pluginServer6) ValidateProviderConfig(context.Context, *tfplugin6.ValidateProviderConfig_Request) (*tfplugin6.ValidateProviderConfig_Response, error) {
	return &tfplugin6.ValidateProviderConfig_Response{}, nil
}

// ValidateResourceConfig implements tfplugin6.ProviderServer.
func (p *pluginServer6) ValidateResourceConfig(context.Context, *tfplugin6.ValidateResourceConfig_Request) (*tfplugin6.ValidateResourceConfig_Response, error) {
	return nil, status.Error(codes.Unimplemented, "provider does not offer any resource types")
}

var serverCapabilities6 = &tfplugin6.ServerCapabilities{
	GetProviderSchemaOptional: true,
}
