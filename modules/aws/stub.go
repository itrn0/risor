//go:build !aws
// +build !aws

package aws

import (
	"github.com/itrn0/risor/object"
)

func Module() *object.Module {
	return nil
}
