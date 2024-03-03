package main

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestProviderUpper(t *testing.T) {
	p := newProvider()
	fn := p.CallStub("upper")

	tests := []struct {
		input cty.Value
		want  cty.Value
	}{
		{
			cty.StringVal("hello"),
			cty.StringVal("HELLO"),
		},
	}

	for _, test := range tests {
		t.Run(test.input.GoString(), func(t *testing.T) {
			got, err := fn(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !test.want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
