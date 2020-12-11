module github.com/FleekHQ/space-daemon

go 1.14

replace github.com/textileio/go-threads => github.com/FleekHQ/go-threads v1.0.1-0.20201028195307-d9371c20fe66

replace github.com/textileio/textile/v2 => github.com/FleekHQ/textile/v2 v2.0.0-20201127024116-cee5aaade92c

replace github.com/libp2p/go-libp2p-pubsub => github.com/libp2p/go-libp2p-pubsub v0.3.2

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.6.1

replace github.com/libp2p/go-libp2p => github.com/libp2p/go-libp2p v0.10.3

replace github.com/libp2p/go-libp2p-swarm => github.com/libp2p/go-libp2p-swarm v0.2.8

require (
	bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc
	github.com/99designs/keyring v1.1.5
	github.com/alecthomas/jsonschema v0.0.0-20191017121752-4bb6e3fae4f2
	github.com/blevesearch/bleve v1.0.12
	github.com/creamdog/gonfig v0.0.0-20160810132730-80d86bfb5a37
	github.com/cznic/b v0.0.0-20181122101859-a26611c4d92d // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/cznic/strutil v0.0.0-20181122101858-275e90344537 // indirect
	github.com/dgraph-io/badger v1.6.2
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/hsanjuan/ipfs-lite v1.1.17 // indirect
	github.com/improbable-eng/grpc-web v0.13.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs v0.7.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-config v0.10.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-path v0.0.8 // indirect
	github.com/ipfs/go-unixfs v0.2.4
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/keybase/go-kext v0.0.0-20200218013902-e4a86908886a
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.14
	github.com/odeke-em/go-utils v0.0.0-20170224015737-e8ebaed0777a
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/opentracing/opentracing-go v1.2.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1 // indirect
	github.com/radovskyb/watcher v1.0.7
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	github.com/textileio/dcrypto v0.0.1
	github.com/textileio/go-threads v1.0.1
	github.com/textileio/textile/v2 v2.1.7
	github.com/tyler-smith/go-bip39 v1.0.2
	github.com/uber/jaeger-client-go v2.23.1+incompatible
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20201006153459-a7d1128ccaa0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20201113135734-0a15ea8d9b02
	google.golang.org/genproto v0.0.0-20200702021140-07506425bd67
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.20.5
	gotest.tools v2.2.0+incompatible
)
