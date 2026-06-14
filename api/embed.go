// Package api holds the hand-maintained OpenAPI 3.1 contract for the
// service. The spec lives next to this file (openapi.yaml) and is embedded
// into the binary so it can be served at /openapi.yaml without any runtime
// filesystem dependency.
//
// The embed directive must live inside the package directory tree, which is
// the only reason this file exists — internal/docs imports OpenAPISpec from
// here rather than trying to reach across directories with go:embed.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
