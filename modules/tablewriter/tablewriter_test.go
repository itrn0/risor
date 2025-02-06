package tablewriter

import (
	"bytes"
	"context"
	"testing"

	"github.com/itrn0/risor/object"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	m := Module()
	require.NotNil(t, m)

	fnObj, ok := m.GetAttr("writer")
	require.True(t, ok)
	fn, ok := fnObj.(*object.Builtin)
	require.True(t, ok)

	result := fn.Call(context.Background())
	require.NotNil(t, result)
	_, ok = result.(*Writer)
	require.True(t, ok)
}

func TestWriter(t *testing.T) {
	buf := new(bytes.Buffer)
	bufObj := object.NewBuffer(buf)
	w := CreateWriter(context.Background(), bufObj)
	require.NotNil(t, w)

	rows := object.NewList(
		[]object.Object{
			object.NewList([]object.Object{
				object.NewString("a"),
				object.NewString("b"),
				object.NewString("c"),
			}),
			object.NewList([]object.Object{
				object.NewString("1"),
				object.NewString("2"),
				object.NewString("3"),
			}),
		},
	)

	opts := object.NewMap(map[string]object.Object{
		"header": object.NewList([]object.Object{
			object.NewString("H1"),
			object.NewString("H2"),
			object.NewString("H3"),
		}),
		"footer": object.NewList([]object.Object{
			object.NewString("F1"),
			object.NewString("F2"),
			object.NewString("F3"),
		}),
		"alignment":           object.NewInt(int64(tablewriter.ALIGN_RIGHT)),
		"row_separator":       object.NewString("="),
		"header_line":         object.NewBool(true),
		"auto_format_headers": object.NewBool(true),
	})

	r := Render(context.Background(), rows, opts, bufObj)
	require.NotNil(t, r)

	require.Equal(t, `+====+====+====+
| H1 | H2 | H3 |
+====+====+====+
|  a |  b |  c |
|  1 |  2 |  3 |
+====+====+====+
| F1 | F2 | F3 |
+====+====+====+
`, buf.String())
}
