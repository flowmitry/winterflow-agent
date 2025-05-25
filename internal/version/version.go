package version

import (
	"strconv"
	"strings"
)

var (
	version = "0.0.0"
)

func GetVersion() string {
	return version
}

func GetNumericVersion() int {
	return ParseNumericVersion(version)
}

func ParseNumericVersion(semVer string) int {
	parts := strings.Split(semVer, ".")
	result := 0
	for _, part := range parts {
		num, _ := strconv.Atoi(part)
		result = result*1000 + num
	}
	return result
}
