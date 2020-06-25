build:
	go build \
	-o bin/space \
	-ldflags \
	"-X 'main.ipfsaddr=${IPFS_ADDR}'" \
	cmd/space-poc/main.go

test:
	go test ./...

proto_gen:
	protoc -I grpc/pb/ grpc/pb/space.proto --go_out=plugins=grpc:grpc/pb