package testutil

import (
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

func init() {
	var isVerbose bool
	for _, arg := range os.Args {
		if arg == "-test.v=true" {
			isVerbose = true
		}
	}

	logrus.SetLevel(logrus.TraceLevel)

	if !isVerbose {
		logrus.StandardLogger().Out = ioutil.Discard
	}
}
