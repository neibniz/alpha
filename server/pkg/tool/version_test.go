package tool

import (
	"testing"

	"github.com/neibniz/alpha/server/pkg/constant"
)

func TestExtractVersionNumber(t *testing.T) {
	versionNumber := ExtractVersionNumber(constant.Version)
	t.Log(versionNumber)
}
