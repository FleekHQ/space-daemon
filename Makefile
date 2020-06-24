build:
	go build -o bin/space cmd/space-poc/main.go

test:
	go test ./...

proto_gen:
	protoc -I grpc/pb/ grpc/pb/space.proto --go_out=plugins=grpc:grpc/pb