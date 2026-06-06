// Isolated keystone module for ADR-002 / spike-concurrent-render-v0170.
// Separate go.mod keeps the CUE version under test off the library's
// production go.mod (which stays at v0.16.1). RETAINED as the reproducible
// evidence ADR-002 cites (run on v0.16.1 → panic, v0.17.0-alpha.1 →
// race-clean + ~2.5x). The recontract change folds this into a permanent
// in-tree -race regression test, after which this module can be deleted.
module spike/crosscontext

go 1.26

require cuelang.org/go v0.17.0-alpha.1

require (
	github.com/cockroachdb/apd/v3 v3.2.3 // indirect
	github.com/emicklei/proto v1.14.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20260420112717-c39628bde8b5 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
