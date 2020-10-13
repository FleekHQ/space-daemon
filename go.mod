module github.com/FleekHQ/space-daemon

go 1.14

replace github.com/ipfs/go-datastore v0.4.4 => github.com/textileio/go-datastore v0.4.5-0.20200728205504-ffeb3591b248

replace github.com/ipfs/go-ds-badger v0.2.4 => github.com/textileio/go-ds-badger v0.2.5-0.20200728212847-1ec9ac5e644c

require (
	bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc
	github.com/99designs/keyring v1.1.5
	github.com/AlecAivazis/survey/v2 v2.0.7 // indirect
	github.com/alangpierce/go-forceexport v0.0.0-20160317203124-8f1d6941cd75 // indirect
	github.com/alecthomas/jsonschema v0.0.0-20191017121752-4bb6e3fae4f2
	github.com/creamdog/gonfig v0.0.0-20160810132730-80d86bfb5a37
	github.com/dgraph-io/badger v1.6.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/color v1.9.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.1
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/improbable-eng/grpc-web v0.13.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs v0.6.1-0.20200817102359-90a573354af2
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-config v0.9.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.4
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/jbenet/goprocess v0.1.4
	github.com/joho/godotenv v1.3.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.14
	github.com/odeke-em/go-utils v0.0.0-20170224015737-e8ebaed0777a
	github.com/odeke-em/go-uuid v0.0.0-20151221120446-b211d769a9aa
	github.com/pkg/errors v0.9.1
	github.com/radovskyb/watcher v1.0.7
	github.com/rakyll/statik v0.1.7
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/textileio/dcrypto v0.0.1
	github.com/textileio/go-threads v0.1.24-0.20200831040109-0d95d73fbdba
	github.com/textileio/textile v1.0.15-0.20200903032519-e084c283e787
	github.com/tyler-smith/go-bip39 v1.0.2
	go.etcd.io/etcd v3.3.22+incompatible
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980
	google.golang.org/genproto v0.0.0-20200702021140-07506425bd67
	google.golang.org/grpc v1.31.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	grpc.go4.org v0.0.0-20170609214715-11d0a25b4919
)
