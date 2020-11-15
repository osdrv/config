package config

// Schema is a pretty flexible structure for schema definitions.
// It might be:
// * a Mapper
// * a Converter
// * a map[string]Schema
type Schema interface{}
