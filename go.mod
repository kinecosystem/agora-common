module github.com/kinecosystem/agora-common

go 1.13

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/DataDog/datadog-go v3.4.1+incompatible
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/aws/aws-sdk-go v1.25.25
	github.com/aws/aws-sdk-go-v2 v0.17.0
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/containerd/continuity v0.0.0-20210315143101-93e15499afd5 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-redis/redis/v7 v7.0.0
	github.com/goburrow/cache v0.1.0
	github.com/golang/protobuf v1.5.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.2
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/kinecosystem/agora-api v0.25.0
	github.com/kinecosystem/go v0.0.0-20191108204735-d6832148266e
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.5.2 // indirect
	github.com/mr-tron/base58 v1.2.0
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc9 // indirect
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stellar/go v0.0.0-20191211203732-552e507ffa37
	github.com/stellar/go-xdr v0.0.0-20200331223602-71a1e6d555f2 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/ybbus/jsonrpc v2.1.2+incompatible
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	mfycheng.dev/retry v1.1.0
)

// This dependency of stellar/go no longer exists; use a forked version of the repo instead.
replace bitbucket.org/ww/goautoneg => github.com/adjust/goautoneg v0.0.0-20150426214442-d788f35a0315
