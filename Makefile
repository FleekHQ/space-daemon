build:
	go build \
	-o bin/space \
	-ldflags \
	"-X 'main.ipfsaddr=${IPFS_ADDR}'\
	-X 'main.mongousr=${MONGO_USR}' \
	-X 'main.mongopw=${MONGO_PW}' \
	-X 'main.spaceapi=${SERVICES_API_URL}' \
	-X 'main.spacehubauth=${SERVICES_HUB_AUTH_URL}' \
	-X 'main.textilehub=${TXL_HUB_TARGET}' \
	-X 'main.textilehubma=${TXL_HUB_MA}' \
	-X 'main.textilethreads=${TXL_THREADS_TARGET}' \
	-X 'main.mongohost=${MONGO_HOST}' \
	-X 'main.mongorepset=${MONGO_REPLICA_SET}'" \
	cmd/space-daemon/main.go

test:
	go test $$(go list ./... | grep -v integration_tests)

integration_test:
	go test -v -p 1 ./integration_tests/...

proto_gen:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb

gen_rest:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb --grpc-gateway_out=logtostderr=true:grpc/pb

## this target requires both protoc-gen-swagger and statik to be installed
gen_rest_swagger:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --swagger_out=logtostderr=true:swagger/ui \
		&& statik -src=swagger/ui -f -dest=swagger -p=bin_ui

gen_all: proto_gen gen_rest gen_rest_swagger
