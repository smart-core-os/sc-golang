module git.vanti.co.uk/smartcore/sc-golang

go 1.14

require (
	git.vanti.co.uk/smartcore/sc-api/go v1.0.0-beta.8
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/iancoleman/strcase v0.1.1
	github.com/mennanov/fieldmask-utils v0.3.3
	github.com/olebedev/emitter v0.0.0-20190110104742-e8d1457e6aee
	go.uber.org/zap v1.15.0
	google.golang.org/grpc v1.30.0
	google.golang.org/protobuf v1.25.0
)

replace git.vanti.co.uk/smartcore/sc-api/go => ../sc-api/go
