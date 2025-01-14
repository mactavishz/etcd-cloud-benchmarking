module git.tu-berlin.de/mactavishz/csb-project-ws2425/api

go 1.23

require (
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1
)

require (
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241223144023-3abc09e42ca8 // indirect
)

replace (
	git.tu-berlin.de/mactavishz/csb-project-ws2425/api => ../api
	git.tu-berlin.de/mactavishz/csb-project-ws2425/client => ../client
	git.tu-berlin.de/mactavishz/csb-project-ws2425/control => ../control
	git.tu-berlin.de/mactavishz/csb-project-ws2425/data-generator => ../data-generator
)
