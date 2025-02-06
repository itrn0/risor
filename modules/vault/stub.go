//go:build !vault
// +build !vault

package vault

import (
	"github.com/itrn0/risor/object"
)

func Module() *object.Module {
	return nil
}
