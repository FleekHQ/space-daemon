# space-poc


Research repository to test and try things
this code later needs to be rewritten or moved to the actual
repositories that will be used for implementation.


## Local Development

### POC Package Structure

Loosely based on these resources:
https://github.com/golang-standards/project-layout

Note: For POC purposes the package structure is not as important but when we do migrate to a real folder we want to
structure as much as possible following standards

* `/api` Folder structure for REST API.
* `/cmd` Entry point directory for all binaries this repo handles. E.g cmd/{binary-name}/main.go
* `/config` Global Config code
* `/core` Directory for core stuff like watcher service and threads watcher
* `/logger` Directory for app logging
* `/examples` Directory playground for general examples and drafts

Other Potential directories to consider adding
`/shared` or  `/common` In case we want to share libs or logic among several binaries like
ipfs logic and other interactions.

### Generating Mocks

Mocks are generated using https://github.com/vektra/mockery.

For Linux it needs to be built from source.

`mockery -name InterfaceToMock -dir path/to/go/files`

### Protobuf

Make sure Protobuf gen tool is in your path:
`export PATH="$PATH:$(go env GOPATH)/bin"`

Run the following go get commands:
`go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway`
`go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger`

Checking the binaries:
`ls $GOPATH/bin`
Should show the following binaries in your path: protoc-gen-go, protoc-gen-grpc-gateway, protoc-gen-swagger

Run the protobuf generation:
`make proto_gen`

Run the REST proxy generation:
`make gen_rest`

To generate REST proxy swagger spec and ui binary generation
`make gen_rest_swagger`

** Ideally you should run `make gen_all` before commiting as this would run all the above three code generations and 
ensure everything is up to date **

NOTE: See here for instructions on Reverse Proxy:
https://github.com/grpc-ecosystem/grpc-gateway

## Running the Space Binary

### Building Generated Code
Before building the space binary, you need to ensure all generated codes are up to date.

Run `make gen_all`.

### Building Space Binary

Set the required env variables:
```
IPFS_ADDR=/ip4/127.0.0.1/tcp/5001
MONGO_PW=mongpw
MONGO_USR=mongousr
MONGO_HOST=mongohost
SERVICES_API_URL=https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev
SERVICES_HUB_AUTH_URL=wss://gqo1oqz055.execute-api.us-west-2.amazonaws.com/dev
TXL_HUB_TARGET=textile-hub-dev.fleek.co:3006
TXL_THREADS_TARGET=textile-hub-dev.fleek.co:3006
```

Run `make`. This will put the binary in the `bin` folder.

### Running

Binary should run in a folder with a space.json config file with the following settings:
```json
{
  "space": {
    "textileHubTarget": "textile-hub-dev.fleek.co:3006",
    "textileThreadsTarget": "textile-hub-dev.fleek.co:3006",
    "rpcPort": 9999,
    "storePath": "~/.fleek-space"
  }
}
```

If you still have issues, try setting the env var SPACE_APP_DIR
`export SPACE_APP_DIR=/path/to/the/space/binary`

## Debugging Space Binary
The following flags can be run with the binary to output profiling files for debugging.
Flags support a full path to a file.
`-cpuprofile cpu.prof -memprofile mem.prof`

By default, the binary runs in debug mode (this may change after release) and it boots a pprof
server in localhost:6060. See docs how to interact with pprof server here: https://github.com/google/pprof/blob/master/doc/README.md

To disable debug mode add this flag to binary arguments
`-debug=false`


