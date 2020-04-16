package metrics

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var (
	ctorMu sync.Mutex
	ctors  = make(map[string]ClientCtor)
)

// A ClientCtor creates a metrics client using the provided config.
type ClientCtor func(config *ClientConfig) (Client, error)

// RegisterClientCtor registers a ClientCtor for the specified client type.
func RegisterClientCtor(clientType string, ctr ClientCtor) {
	ctorMu.Lock()
	defer ctorMu.Unlock()

	_, exists := ctors[clientType]
	if exists {
		panic(fmt.Sprintf("metrics.ClientCtor already registered for clientType '%s'", clientType))
	}

	ctors[clientType] = ctr
}

// CreateClient creates a Client using the ClientCtor of the requested type, if
// one has been registered. If no constructor has been registered with the specified
// type, an error is thrown.
func CreateClient(clientType string, config *ClientConfig) (Client, error) {
	ctorMu.Lock()
	ctor, ok := ctors[clientType]
	ctorMu.Unlock()
	if !ok {
		return nil, errors.Errorf("ClientCtor with type %s not found", clientType)
	}
	return ctor(config)
}
