package version

import (
	"winterflow-agent/pkg/semver"
)

var (
	version = "0.0.0"
)

func GetVersion() string {
	return version
}

func GetNumericVersion() int {
	return semver.GetNumericVersion(version)
}
