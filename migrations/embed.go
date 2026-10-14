package migrations

import _ "embed"

// InitialSchema is packaged with the service so server deployments need no SQL files.
//
//go:embed 001_init_schema.sql
var InitialSchema string
