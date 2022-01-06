package electric

// PREREQUISITE: protoc-gen-router is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-router`
//go:generate protoc -I ../../../.protomod -I ../../../.protomod/github.com/smart-core-os/sc-api/protobuf/ -I . --router_out=../../../ github.com/smart-core-os/sc-api/protobuf/traits/electric.proto ./memory_settings.proto
