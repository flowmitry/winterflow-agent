package semver

import (
	"strconv"
	"strings"
)

func GetNumericVersion(semVer string) int {
	parts := strings.Split(semVer, ".")
	result := 0
	for _, part := range parts {
		num, _ := strconv.Atoi(part)
		result = result*1000 + num
	}
	return result
}
