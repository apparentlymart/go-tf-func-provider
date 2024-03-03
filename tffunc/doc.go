// Package tffunc is a small library for building providers for Terraform (and
// other software that implements its provider protocol) that exclusively offer
// functions, and do not offer other concepts like resource types.
//
// Typical providers are focused primarily on resource types, and might offer
// a few functions to support those resource types. This library is not suitable
// for those providers, and so you should use a different library such as the
// HashiCorp Terraform Plugin Framework if you wish to build a fully-fledged
// provider.
//
// This library deals with the simpler case of a utility provider that exists
// only to extend the Terraform language with new functions.
//
// This is not a HashiCorp project.
package tffunc
