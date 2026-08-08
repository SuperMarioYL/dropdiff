// Package data embeds the curated drop-spec registry shipped with the binary
// so `dropdiff check` works out of the box, before the user ever runs sync.
package data

import _ "embed"

//go:embed registry.yaml
var RegistryYAML []byte
