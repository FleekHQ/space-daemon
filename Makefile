build:
	go build \
	-o bin/space \
	-ldflags \
	"-X 'main.ipfsnodeaddr=${IPFS_NODE_ADDR}' \
	-X 'main.ipfsnodepath=${IPFS_NODE_PATH}' \
	-X 'main.spaceapi=${SERVICES_API_URL}' \
	-X 'main.spacestoragesiteurl=${SPACE_STORAGE_SITE_URL}' \
	-X 'main.vaultapi=${VAULT_API_URL}' \
	-X 'main.vaultsaltsecret=${VAULT_SALT_SECRET}' \
	-X 'main.spacehubauth=${SERVICES_HUB_AUTH_URL}' \
	-X 'main.textilehub=${TXL_HUB_TARGET}' \
	-X 'main.textilehubma=${TXL_HUB_MA}' \
	-X 'main.textilethreads=${TXL_THREADS_TARGET}' \
	-X 'main.textilehubgatewayurl=${TXL_HUB_GATEWAY_URL}' \
    -X 'main.textileuserkey=${TXL_USER_KEY}' \
	-X 'main.textileusersecret=${TXL_USER_SECRET}'" \
	cmd/space-daemon/main.go

test:
	go test ./...

proto_gen:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb

gen_rest:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb --grpc-gateway_out=logtostderr=true:grpc/pb


gen_all: proto_gen gen_rest

## runs jaeger tracing server, should be used when trace is enabled on daemon
jaegar:
	docker run \
		--rm \
		--name jaeger \
		-p 6831:6831/udp \
		-p 16686:16686 \
		jaegertracing/all-in-one:latest
