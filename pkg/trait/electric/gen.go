package electric

// PREREQUISITE: protocmod is on PATH, i.e. `go install git.vanti.co.uk/vanti-incubator/protomod`
// PREREQUISITE: protoc-gen-router is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-router`
// PREREQUISITE: protoc-gen-wrapper is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-wrapper`
//go:generate protomod protoc -- -I ../../.. --router_out=../../.. pkg/trait/electric/memory_settings.proto github.com/smart-core-os/sc-api/protobuf/traits/electric.proto
//go:generate protomod protoc -- -I ../../.. --wrapper_out=../../.. pkg/trait/electric/memory_settings.proto github.com/smart-core-os/sc-api/protobuf/traits/electric.proto
