module github.com/kinecosystem/agora-common

go 1.13

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/DataDog/datadog-go v3.4.1+incompatible
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/aws/aws-sdk-go v1.25.25
	github.com/aws/aws-sdk-go-v2 v0.17.0
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/containerd/continuity v0.0.0-20190827140505-75bee3e2ccb6 // indirect
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-redis/redis/v7 v7.0.0
	github.com/goburrow/cache v0.1.0
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.9.0
	github.com/kinecosystem/go v0.0.0-20191108204735-d6832148266e
	github.com/lib/pq v1.5.2 // indirect
	github.com/mr-tron/base58 v1.2.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.5.0
	github.com/stellar/go v0.0.0-20191211203732-552e507ffa37
	github.com/stellar/go-xdr v0.0.0-20200331223602-71a1e6d555f2 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/ybbus/jsonrpc v2.1.2+incompatible
	go.etcd.io/etcd v3.3.25+incompatible
	golang.org/x/net v0.0.0-20191119073136-fc4aabc6c914 // indirect
	google.golang.org/genproto v0.0.0-20191115221424-83cc0476cb11 // indirect
	google.golang.org/grpc v1.25.1
	gotest.tools v2.2.0+incompatible // indirect
	mfycheng.dev/retry v1.1.0
)

// This dependency of stellar/go no longer exists; use a forked version of the repo instead.
replace bitbucket.org/ww/goautoneg => github.com/adjust/goautoneg v0.0.0-20150426214442-d788f35a0315
