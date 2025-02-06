//go:build !semver
// +build !semver

package semver

import (
	"github.com/itrn0/risor/object"
)

func Module() *object.Module {
	return nil
}
