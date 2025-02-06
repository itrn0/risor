package object

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/itrn0/risor/op"
)

type Color struct {
	*base
	value color.Color
}

func (c *Color) Inspect() string {
	return c.String()
}

func (c *Color) Type() Type {
	return COLOR
}

func (c *Color) Value() color.Color {
	return c.value
}

func (c *Color) GetAttr(name string) (Object, bool) {
	switch name {
	case "rgba":
		return NewBuiltin("color.rgba",
			func(ctx context.Context, args ...Object) Object {
				if len(args) != 0 {
					return NewArgsError("color.rgba", 0, len(args))
				}
				r, g, b, a := c.value.RGBA()
				return NewList([]Object{
					NewInt(int64(r)),
					NewInt(int64(g)),
					NewInt(int64(b)),
					NewInt(int64(a)),
				})
			}), true
	}
	return nil, false
}

func (c *Color) SetAttr(name string, value Object) error {
	return TypeErrorf("type error: color object has no attribute %q", name)
}

func (c *Color) Interface() interface{} {
	return c.value
}

func (c *Color) String() string {
	r, g, b, a := c.value.RGBA()
	return fmt.Sprintf("color(r=%d g=%d b=%d a=%d)", r, g, b, a)
}

func (c *Color) Equals(other Object) Object {
	if c == other {
		return True
	}
	return False
}

func (c *Color) RunOperation(opType op.BinaryOpType, right Object) Object {
	return TypeErrorf("type error: unsupported operation for color: %v ", opType)
}

func (c *Color) MarshalJSON() ([]byte, error) {
	r, g, b, a := c.value.RGBA()
	return json.Marshal(struct {
		R uint32 `json:"r"`
		G uint32 `json:"g"`
		B uint32 `json:"b"`
		A uint32 `json:"a"`
	}{
		R: r,
		G: g,
		B: b,
		A: a,
	})
}

func NewColor(c color.Color) *Color {
	return &Color{value: c}
}
