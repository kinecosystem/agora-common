package app

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/pkg/errors"
)

var (
	ctrMu sync.Mutex
	ctrs  = make(map[string]FileLoaderCtr)
)

// RegisterFileLoader registers a FileLoader for the specified scheme.
func RegisterFileLoader(scheme string, ctr FileLoaderCtr) {
	ctrMu.Lock()
	defer ctrMu.Unlock()

	_, exists := ctrs[scheme]
	if exists {
		panic(fmt.Sprintf("FileLoader already registered for scheme '%s'", scheme))
	}

	ctrs[scheme] = ctr
}

// FileLoaderCtr constructs a FileLoader.
type FileLoaderCtr func() (FileLoader, error)

// FileLoader loads files at a specified URL.
type FileLoader interface {
	Load(url *url.URL) ([]byte, error)
}

// LoadFile loads a file at the specified URL using the corresponding
// registered FileLoader. If no scheme is specified, LocalLoader is used.
func LoadFile(fileURL string) ([]byte, error) {
	ctrMu.Lock()
	defer ctrMu.Unlock()

	u, err := url.Parse(fileURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid file url %s", fileURL)
	}

	ctr, exists := ctrs[u.Scheme]
	if !exists {
		return nil, errors.Errorf("no file loader for %s", u.Scheme)
	}

	l, err := ctr()
	if err != nil {
		return nil, errors.Wrapf(err, "failed get loader for '%s'", fileURL)
	}

	return l.Load(u)
}
