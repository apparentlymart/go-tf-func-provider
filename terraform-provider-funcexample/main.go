// terraform-provider-funcexample is an example Terraform provider that
// exports some contrived functions for illustrative purposes only.
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	p := newProvider()
	err := p.Serve(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting plugin: %s", err)
		os.Exit(1)
	}
}
