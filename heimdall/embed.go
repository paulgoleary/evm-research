package heimdall

import "embed"

//go:embed data/checkpoint.ndjson
var fchk embed.FS
var checkPointData, _ = fchk.ReadFile("data/checkpoint.ndjson")

//go:embed data/side-tx1.json
var fstx embed.FS
var sideTxData, _ = fstx.ReadFile("data/side-tx1.json")
