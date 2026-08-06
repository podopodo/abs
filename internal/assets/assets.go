package assets

// @ACP O ACP.PROTOCOL

import _ "embed"

//go:embed protocol.md
var Protocol string

//go:embed context.md
var RootContext string

//go:embed config.json
var ConfigJSON string
