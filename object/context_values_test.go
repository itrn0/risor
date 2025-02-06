package object

import (
	"context"
	"testing"

	"github.com/itrn0/risor/compiler"
	"github.com/stretchr/testify/require"
)

func TestContextCallFunc(t *testing.T) {
	callFunc, ok := GetCallFunc(context.Background())
	require.False(t, ok)
	require.Nil(t, callFunc)

	ctx := WithCallFunc(context.Background(),
		func(ctx context.Context, fn *Function, args []Object) (Object, error) {
			return NewInt(42), nil
		})
	callFunc, ok = GetCallFunc(ctx)
	require.True(t, ok)
	require.NotNil(t, callFunc)

	result, err := callFunc(context.Background(),
		NewFunction(compiler.NewFunction(compiler.FunctionOpts{})),
		[]Object{})
	require.Nil(t, err)
	require.Equal(t, NewInt(42), result)
}

func TestContextSpawnFunc(t *testing.T) {
	fn, ok := GetSpawnFunc(context.Background())
	require.False(t, ok)
	require.Nil(t, fn)

	ctx := WithSpawnFunc(context.Background(),
		func(ctx context.Context, fn Callable, args []Object) (*Thread, error) {
			return &Thread{}, nil
		})
	fn, ok = GetSpawnFunc(ctx)
	require.True(t, ok)
	require.NotNil(t, fn)

	result, err := fn(context.Background(),
		NewFunction(compiler.NewFunction(compiler.FunctionOpts{})),
		[]Object{})
	require.Nil(t, err)
	require.IsType(t, &Thread{}, result)
}

func TestContextCloneCallFunc(t *testing.T) {
	fn, ok := GetCloneCallFunc(context.Background())
	require.False(t, ok)
	require.Nil(t, fn)

	ctx := WithCloneCallFunc(context.Background(),
		func(ctx context.Context, fn *Function, args []Object) (Object, error) {
			return NewInt(42), nil
		})
	fn, ok = GetCloneCallFunc(ctx)
	require.True(t, ok)
	require.NotNil(t, fn)

	result, err := fn(context.Background(),
		NewFunction(compiler.NewFunction(compiler.FunctionOpts{})),
		[]Object{})
	require.Nil(t, err)
	require.IsType(t, NewInt(42), result)
}
