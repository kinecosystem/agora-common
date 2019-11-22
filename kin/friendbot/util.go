package friendbot

import (
	"encoding/json"
	"net/http"

	"github.com/kinecosystem/go/clients/horizon"
	"github.com/kinecosystem/go/support/errors"
)

// decodeResponse decodes a response from a Horizon server. This is defined in kinecosystem/go but in an internal file
// (https://github.com/kinecosystem/go/blob/master/clients/horizon/internal.go#L10), so it is included here for use
// within this package.
func decodeResponse(resp *http.Response, object interface{}) (err error) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	// only 2xx responses will be decoded - any redirections should be followed prior to calling decodeResponse
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		horizonError := &horizon.Error{
			Response: resp,
		}
		decodeError := decoder.Decode(&horizonError.Problem)
		if decodeError != nil {
			return errors.Wrap(decodeError, "error decoding horizon.Problem")
		}
		return horizonError
	}

	return decoder.Decode(&object)
}
