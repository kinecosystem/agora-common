package version

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"github.com/kinecosystem/agora-common/headers"
)

type KinVersion uint16

const (
	KinVersionUnknown KinVersion = iota
	KinVersionReserved
	KinVersion2
	KinVersion3
	KinVersion4
)

const (
	KinVersionHeader        = "kin-version"
	DesiredKinVersionHeader = "desired-kin-version"
	minVersion              = KinVersion2
	maxVersion              = KinVersion4
	defaultVersion          = KinVersion3
)

// GetCtxKinVersion determines which version of Kin to use based on the headers in the provided context.
func GetCtxKinVersion(ctx context.Context) (v KinVersion, err error) {
	val, err := headers.GetASCIIHeaderByName(ctx, KinVersionHeader)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get kin version header")
	}

	if len(val) == 0 {
		return defaultVersion, nil
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse integer version from string")
	}

	if i < int(minVersion) || i > int(maxVersion) {
		return 0, errors.Wrap(err, "invalid kin version")
	}

	return KinVersion(i), nil
}

// GetCtxDesiredVersion determines which version of Kin the requestor whiches to have enforced.
func GetCtxDesiredVersion(ctx context.Context) (v KinVersion, err error) {
	val, err := headers.GetASCIIHeaderByName(ctx, DesiredKinVersionHeader)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get desired kin version header")
	}

	if len(val) == 0 {
		return 0, errors.New("no desired kin version set")
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse integer version from string")
	}

	if i < int(minVersion) || i > int(maxVersion) {
		return 0, errors.Wrap(err, "invalid desired kin version")
	}

	return KinVersion(i), nil
}
