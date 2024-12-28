.PHONY: gen clean

gen:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		**/*.proto

clean:
	rm -f **/*.pb.go
