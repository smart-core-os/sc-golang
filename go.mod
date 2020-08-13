module sc-golang

go 1.14

require (
	git.vanti.co.uk/smartcore/sc-api/go v1.0.0-beta.4
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	go.uber.org/zap v1.15.0
	google.golang.org/grpc v1.30.0
)

replace git.vanti.co.uk/smartcore/sc-api/go => ../sc-api/go
