module github.com/FleekHQ/space-daemon

go 1.14

replace github.com/textileio/go-threads => github.com/FleekHQ/go-threads v1.0.1-0.20201028195307-d9371c20fe66

replace github.com/libp2p/go-libp2p-kad-dht => github.com/libp2p/go-libp2p-kad-dht v0.8.3

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.6.1

replace github.com/libp2p/go-libp2p => github.com/libp2p/go-libp2p v0.10.3

replace github.com/libp2p/go-libp2p-swarm => github.com/libp2p/go-libp2p-swarm v0.2.8

require (
	bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc
	github.com/99designs/keyring v1.1.5
	github.com/AlecAivazis/survey/v2 v2.0.7 // indirect
	github.com/alangpierce/go-forceexport v0.0.0-20160317203124-8f1d6941cd75 // indirect
	github.com/alecthomas/jsonschema v0.0.0-20191017121752-4bb6e3fae4f2
	github.com/blevesearch/bleve v1.0.12
	github.com/blevesearch/cld2 v0.0.0-20200327141045-8b5f551d37f5 // indirect
	github.com/creamdog/gonfig v0.0.0-20160810132730-80d86bfb5a37
	github.com/cznic/b v0.0.0-20181122101859-a26611c4d92d // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/cznic/strutil v0.0.0-20181122101858-275e90344537 // indirect
	github.com/dgraph-io/badger v1.6.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/color v1.9.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/ikawaha/kagome.ipadic v1.1.2 // indirect
	github.com/improbable-eng/grpc-web v0.13.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs v0.7.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-config v0.9.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.4
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kyokomi/emoji v2.2.4+incompatible // indirect
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.3 // indirect
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.14
	github.com/odeke-em/go-utils v0.0.0-20170224015737-e8ebaed0777a
	github.com/odeke-em/go-uuid v0.0.0-20151221120446-b211d769a9aa // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/radovskyb/watcher v1.0.7
	github.com/rakyll/statik v0.1.7
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tebeka/snowball v0.4.2 // indirect
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	github.com/textileio/dcrypto v0.0.1
	github.com/textileio/go-threads v1.0.1
	github.com/textileio/textile v1.0.14 // indirect
	github.com/textileio/textile/v2 v2.1.7
	github.com/tyler-smith/go-bip39 v1.0.2
	github.com/uber/jaeger-client-go v2.23.1+incompatible
	go.etcd.io/etcd v3.3.22+incompatible // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20201113135734-0a15ea8d9b02
	google.golang.org/genproto v0.0.0-20200702021140-07506425bd67
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.20.5
	gotest.tools v2.2.0+incompatible
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919 // indirect
)
