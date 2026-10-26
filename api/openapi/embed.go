// Package openapi embeds the hand-written OpenAPI 3.1 spec. The API serves it
// at /api/openapi/openapi.yaml and tests check it against the route table.
package openapi

import "embed"

// FS holds openapi.yaml plus the per-feature paths/ and schemas/ files.
//
//go:embed openapi.yaml paths schemas
var FS embed.FS
