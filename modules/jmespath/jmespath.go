package jmespath

import (
	"context"
	_ "embed"

	"github.com/itrn0/risor/object"
	"github.com/jmespath-community/go-jmespath/pkg/api"
	"github.com/jmespath-community/go-jmespath/pkg/parsing"
)

func Jmespath(ctx context.Context, args ...object.Object) object.Object {
	numArgs := len(args)

	if numArgs != 2 {
		return object.NewArgsError("jmespath", 2, numArgs)
	}

	switch args[0].(type) {
	case *object.Map,
		*object.List,
		*object.String,
		*object.Int,
		*object.Float,
		*object.Bool,
		*object.Set,
		*object.NilType:
	default:
		return object.TypeErrorf("type error: jmespath() cannot operate on %s", args[0].Type())
	}

	data := args[0].Interface()

	expression, argsErr := object.AsString(args[1])
	if argsErr != nil {
		return argsErr
	}

	if _, err := parsing.NewParser().Parse(expression); err != nil {
		if syntaxError, ok := err.(parsing.SyntaxError); ok {
			return object.Errorf("%s\n%s", syntaxError, syntaxError.HighlightLocation())
		}
		return object.NewError(err)
	}
	result, err := api.Search(expression, data)
	if argsErr != nil {
		return object.NewError(err)
	}

	return object.FromGoType(result)
}

func Builtins() map[string]object.Object {
	return map[string]object.Object{
		"jmespath": object.NewBuiltin("jmespath", Jmespath),
	}
}
