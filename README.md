# Space Daemon

Space Daemon is a wrapper built in Go around awesome IPFS tools so that you can have start coding a decentralized desktop app as fast as possible. It's built on top of Textile Threads and Buckets. Out of the box it includes:

- A running local instance of [Textile Threads](https://github.com/textileio/go-threads).

- Interfaces to create local private, encrypted buckets.

- Interfaces for sharing those buckets and the files within.

- Identity service so that sharing can be done through usernames or emails.

- FUSE for drive mounting, so that the files can be explored natively in your OS.

- Key management.

Note: This project is in active development, so it might change its API until it reaches a stable version.

## Installation

By default, Space Daemon connects to hosted services provided by Fleek. This should be good if you just want to get it running quickly. However, if you want to connect to your own services, read the [Modules Section](https://github.com/FleekHQ/space-daemon#Modules).

### Downloading the binary

Check out the releases [here](https://github.com/FleekHQ/space-daemon/releases). You can download the latest version for your OS and you should be good to go.

If you want to run Space Daemon by source, check out [this section](https://github.com/FleekHQ/space-daemon#Running)

## Usage

Space Daemon provides a gRPC interface. You can read its proto schema [here](https://github.com/FleekHQ/space-daemon/blob/master/grpc/proto/space.proto). It contains methods to:

- Create files and directories

- List files and directories

- Creating buckets

- Sharing buckets

- Creating identities

You can also use the JavaScript client here [https://github.com/FleekHQ/space-client](https://github.com/FleekHQ/space-client)

This can be useful if, for example, you are building a web app that needs to interact with a user's locally running Space Daemon.

## Modules

Space Daemon requires a few modules to run successfully. If you downloaded the binary, you don't have to worry about this since it will be connecting to our services. It's good to understand what's happening behind the scenes though.

### IPFS Node

All encrypted files are stored in an IPFS node. For convenience, Space Daemon runs an embedded node within the daemon that can be configured as well as the option to specify an external node to connect to. 

If you have your own node outside of the daemon, then set the flag `-ipfsnode` to `false`.  This will not spin up an embedded node. You can then connect to your external node by providing the `-ipfsaddr` flag (e.g. `-ipfsaddr=/ip4/127.0.0.1/tcp/5001`).

In the case you are running the embedded IPFS node, you can further configure the listen address and data directory by setting these flags respectively: `-ipfsnodeaddr` and `-ipfsnodepath`.

### Textile Hub

Required for sharing files between users and backing it up. It stores all backed up files encrypted using a set of keys so that only you, and people you share files with, can read the data. We host our own instance of the Textile Hub, and by default, Space Daemon will conect to it. It can be customized by providing the `-textilehub` flag and `-textilethreads` flag.

If you want to host your own Textile Hub node, you can [read its documentation here](https://github.com/textileio/textile)

### Space Services

We provide hosted alternatives for these services. You can deploy your own by following the instructions in its repo:

[https://github.com/fleekHQ/space-services](https://github.com/fleekHQ/space-services)

#### Identity

These are centralized services that are optional, but offer additional convenience. Used mainly for identity. By using these services, you can allow users to claim usernames, so that Space Daemon can know the public key of a given username and in that way share files via username without having to input public keys directly.

#### Authentication

Our hosted Textile Hub requires authentication via public key for logging in. This service sends a challenge to Space Daemon, which signs the challenge with the private key of the user and in that way our hosted Textile Hub can allow the user to store data.

## Running from source

After cloning this repo, you can run it from source by running `go run ./cmd/space-daemon -dev`. Consider that you will need the following environment variables exported in your system:

```
IPFS_ADDR=[Your IPFS node address]
SERVICES_API_URL=[The URL where Space Services API is located]
VAULT_API_URL=[The URL where Space Vault API is located]
VAULT_SALT_SECRET=[A random string used for kdf functions before storing keys to the vault]
SERVICES_HUB_AUTH_URL=[The URL where Space Services Textile Hub Authorizer is located]
TXL_HUB_TARGET=[The URL of the Textile Hub]
TXL_HUB_MA=[The multiaddress for the Textile hub]
TXL_THREADS_TARGET=[The URL of the Textile Hub where Threads are hosted, can be the same that TXL_HUB_TARGET]

# NOTE: the following are required temporarily and will be removed once hub auth wrapper is setup
TXL_USER_KEY=[Space level key for hub access]
TXL_USER_SECRET=[Space level secret for hub access]
```

Alternatively, you can run `make` to compile the binary. Make sure you have these environment variables exposed though. You can see some example environment variables in `.env.example`.

## Contributing

We are happy to receive issues and review pull requests. Please make sure to write tests for the code you are introducing and make sure it doesn't break already passing tests.

Read the following sections for an introduction into the code.

### Package Structure

Loosely based on these resources:
https://github.com/golang-standards/project-layout


- `/grpc` Folder structure for gRPC and REST API.
- `/cmd` Entry point directory for all binaries this repo handles. E.g cmd/{binary-name}/main.go
- `/config` Global Config code
- `/core` Directory for the core objects of the package
- `/logger` Directory for app logging
- `/examples` Directory playground for general examples and drafts

### Main classes

- `ipfs`: contains utils for general IPFS operations.
- `keychain`: manages user public/private key pair.
- `libfuse`: interoperates with FUSE for mounting drives.
- `space`: contains the main integration from the services to the final Textile or FS operations.
- `store`: contains a wrapper around a local db.
- `sync`: keeps track of open files so that the updates get pushed to IPFS
- `textile`: wrapper around Textile booting and operations

### Generating Mocks

Mocks are generated using https://github.com/vektra/mockery.

For Linux it needs to be built from source.

`mockery --name InterfaceToMock --dir path/to/go/files`

### Protobuf

If you update the gRPC API, you need to regenerate the Protobuf file.

You will need to install the following binaries in your Go path:

- `go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway`

Checking the binaries:
`ls $GOPATH/bin`
Should show the following binaries in your path: protoc-gen-go, protoc-gen-grpc-gateway

Run the protobuf generation:
`make proto_gen`

Run the REST proxy generation:
`make gen_rest`

** Ideally you should run `make gen_all` before commiting as this would run all the above three code generations and
ensure everything is up to date **

NOTE: See here for instructions on Reverse Proxy:
https://github.com/grpc-ecosystem/grpc-gateway

### Debugging, Profiling and Tracing

The following flags can be run with the binary to output profiling files for debugging.
Flags support a full path to a file.
`-cpuprofile cpu.prof -memprofile mem.prof`

By default, the binary runs in debug mode (this may change after release) and it boots a pprof
server in localhost:6060. See docs how to interact with pprof server here: https://github.com/google/pprof/blob/master/doc/README.md

To disable debug mode add this flag to binary arguments
`-debug=false`

To enable trace in the daemon, pass `-trace` to the binary arguments. The daemon uses [jaegar](https://www.jaegertracing.io/) 
for collecting trace information. Run `make jaegar` to quickly start a jaeger agent that collects the daemons trace information.
You can `http://localhost:16686/` to explore the web ui for traces collected.

### CI Secrets

Secrets are set by adding them in Github and then specifying them in `release.yml`. Secrets can be constant across environment/stages or be stage specific.

If specified, the release file will dynamically generate the secret name based on the stage by adding a `_DEV` or `_PRD` suffix to the secret name only for the specificed environment variable. It will always use `_PRD` unless the tag ends in `-dev`.  So for example tag `v0.0.15` will use PRD values, while `v0.0.15-dev` will use DEV values.

Stage specific secret names will only be used for secrets in `release.yml` that point to the step output instead of the secret name directly (i.e., `SERVICES_API_URL: ${{ secrets[steps.secretnames.outputs.SERVICES_API_URL] }}` instead of `SERVICES_API_URL: ${{ secrets.SERVICES_API_URL }}`.

So to add a new secret:
* If it's not stage specific then add the secret in GH with no suffix and in `release.yml`, refer to it based on the secret name.
* If it is stage specific, then create the 2 secrets in GH (ending in `_PRD` and `_DEV`), add the entry in step `secretnames`, and make sure the secret name in the next step points to the step output
