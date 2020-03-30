package app

import (
	"io/ioutil"
	"net/url"
)

// LocalLoader is a FileLoader that loads file from the local filesystem.
type LocalLoader struct{}

// Load implements Loader.Load.
func (l LocalLoader) Load(url *url.URL) ([]byte, error) {
	return ioutil.ReadFile(url.Path)
}

func init() {
	ctr := func() (FileLoader, error) {
		return &LocalLoader{}, nil
	}

	RegisterFileLoaderCtor("", ctr)
	RegisterFileLoaderCtor("file", ctr)
}
