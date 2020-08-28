build:
	go build \
	-o bin/space \
	-ldflags \
	"-X 'main.ipfsnodeaddr=${IPFS_NODE_ADDR}' \
	-X 'main.ipfsnodepath=${IPFS_NODE_PATH}' \
	-X 'main.mongousr=${MONGO_USR}' \
	-X 'main.mongopw=${MONGO_PW}' \
	-X 'main.spaceapi=${SERVICES_API_URL}' \
	-X 'main.vaultapi=${VAULT_API_URL}' \
	-X 'main.vaultsaltsecret=${VAULT_SALT_SECRET}' \
	-X 'main.spacehubauth=${SERVICES_HUB_AUTH_URL}' \
	-X 'main.textilehub=${TXL_HUB_TARGET}' \
	-X 'main.textilehubma=${TXL_HUB_MA}' \
	-X 'main.textilethreads=${TXL_THREADS_TARGET}' \
  -X 'main.textileuserkey=${TXL_USER_KEY}' \
	-X 'main.textileusersecret=${TXL_USER_SECRET}' \
	-X 'main.mongohost=${MONGO_HOST}' \
	-X 'main.mongorepset=${MONGO_REPLICA_SET}'" \
	cmd/space-daemon/main.go

test:
	go test ./...

proto_gen:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb

gen_rest:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --go_out=plugins=grpc:grpc/pb --grpc-gateway_out=logtostderr=true:grpc/pb

## this target requires both protoc-gen-swagger and statik to be installed
gen_rest_swagger:
	protoc -I grpc/pb/ -I grpc/proto/ -I./devtools/googleapis grpc/proto/space.proto --swagger_out=logtostderr=true:swagger/ui \
		&& statik -src=swagger/ui -f -dest=swagger -p=bin_ui

gen_all: proto_gen gen_rest gen_rest_swagger

localnet-down:
	docker-compose -p localnet \
		-f docker-compose-localnet.yaml \
		down

localnet-up:
	docker-compose -p localnet \
		-f docker/docker-compose-localnet.yaml \
		up --build -V