package arg_test

import (
	"testing"

	"github.com/itrn0/risor/arg"
	"github.com/itrn0/risor/object"
	"github.com/stretchr/testify/require"
)

func TestRequire(t *testing.T) {
	var err *object.Error

	err = arg.Require(
		"foo",
		1,
		[]object.Object{object.NewInt(1)},
	)
	require.Nil(t, err)

	err = arg.Require(
		"foo",
		1,
		[]object.Object{
			object.NewInt(1),
			object.NewInt(1),
			object.NewInt(1),
		},
	)
	require.NotNil(t, err)
	require.Equal(t, "args error: foo() takes exactly 1 argument (3 given)",
		err.Message().Value())

	err = arg.Require(
		"bar",
		2,
		[]object.Object{object.NewInt(1)},
	)
	require.NotNil(t, err)
	require.Equal(t, "args error: bar() takes exactly 2 arguments (1 given)",
		err.Message().Value())
}

func TestRequireRange(t *testing.T) {
	var err *object.Error

	err = arg.RequireRange(
		"foo",
		1,
		3,
		[]object.Object{object.NewInt(1)},
	)
	require.Nil(t, err)

	err = arg.RequireRange(
		"foo",
		1,
		3,
		[]object.Object{
			object.NewInt(1),
			object.NewInt(1),
			object.NewInt(1),
			object.NewInt(1),
		},
	)
	require.NotNil(t, err)
	require.Equal(t, "args error: foo() takes at most 3 arguments (4 given)",
		err.Message().Value())

	err = arg.RequireRange(
		"foo",
		1,
		3,
		[]object.Object{},
	)
	require.NotNil(t, err)
	require.Equal(t, "args error: foo() takes at least 1 argument (0 given)",
		err.Message().Value())
}
