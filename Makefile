build:
	go build \
	-o bin/space \
	-ldflags \
	"-X 'main.ipfsaddr=${IPFS_ADDR}'\
	-X 'main.mongousr=${MONGO_USR}' \
	-X 'main.mongopw=${MONGO_PW}' \
	-X 'main.spaceapi=${SERVICES_API_URL}' \
	-X 'main.spacehubauth=${SERVICES_HUB_AUTH_URL}' \
	-X 'main.mongohost=${MONGO_HOST}'" \
	cmd/space-poc/main.go

test:
	go test ./...

proto_gen:
	protoc -I grpc/pb/ grpc/pb/space.proto --go_out=plugins=grpc:grpc/pb