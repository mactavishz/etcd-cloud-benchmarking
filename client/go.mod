module git.tu-berlin.de/mactavishz/csb-project-ws2425/client

go 1.23

require (
	git.tu-berlin.de/mactavishz/csb-project-ws2425/api v0.0.0-00010101000000-000000000000
	git.tu-berlin.de/mactavishz/csb-project-ws2425/control v0.0.0-00010101000000-000000000000
	git.tu-berlin.de/mactavishz/csb-project-ws2425/data-generator v0.0.0-00010101000000-000000000000
	go.etcd.io/etcd/api/v3 v3.5.17
	go.etcd.io/etcd/client/v3 v3.5.17
	go.uber.org/zap v1.17.0
	google.golang.org/grpc v1.69.2
)

require (
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.17 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241223144023-3abc09e42ca8 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
)

replace (
	git.tu-berlin.de/mactavishz/csb-project-ws2425/api => ../api
	git.tu-berlin.de/mactavishz/csb-project-ws2425/client => ../client
	git.tu-berlin.de/mactavishz/csb-project-ws2425/control => ../control
	git.tu-berlin.de/mactavishz/csb-project-ws2425/data-generator => ../data-generator
)
