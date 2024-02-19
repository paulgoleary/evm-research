package heimdall

import "embed"

//go:embed data/checkpoint.ndjson
var fchk embed.FS
var checkPointData, _ = fchk.ReadFile("data/checkpoint.ndjson")
