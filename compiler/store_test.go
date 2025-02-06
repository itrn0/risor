package compiler

import (
	"context"
	"fmt"
	"testing"

	"github.com/itrn0/risor/op"
	"github.com/itrn0/risor/parser"
	"github.com/stretchr/testify/require"
)

func compileSource(source string) (*Code, error) {
	program, err := parser.Parse(context.Background(), source)
	if err != nil {
		return nil, err
	}
	opt := WithGlobalNames([]string{"len", "list", "string", "print"})
	code, err := Compile(program, opt)
	if err != nil {
		return nil, err
	}
	return code, nil
}

func TestMarshalCode1(t *testing.T) {
	codeA, err := compileSource(`
	x := 1.0
	y := 2.0
	x + y
	`)
	require.Nil(t, err)
	data, err := MarshalCode(codeA)
	require.Nil(t, err)
	codeB, err := UnmarshalCode(data)
	require.Nil(t, err)
	require.Equal(t, codeA, codeB)
}

func TestMarshalCode2(t *testing.T) {
	codeA, err := compileSource(`
	func test(a, b=2) {
		if a > b {
			return a
		} else {
			return b
		}
	}
	test(1) + test(2, 3)
	`)
	require.Nil(t, err)
	data, err := MarshalCode(codeA)
	require.Nil(t, err)
	codeB, err := UnmarshalCode(data)
	require.Nil(t, err)
	require.Equal(t, codeA, codeB)
}

func TestMarshalCode3(t *testing.T) {
	codeA, err := compileSource(`
	start := 10
	func counter(a) {
		current := a
		return func() {
			current++
			return current
		}
	}
	c := counter(start)
	c()
	`)
	require.Nil(t, err)
	data, err := MarshalCode(codeA)
	require.Nil(t, err)
	fmt.Println(string(data))
	codeB, err := UnmarshalCode(data)
	require.Nil(t, err)
	require.Equal(t, codeA, codeB)
}

func TestMarshalCode4(t *testing.T) {
	codeA, err := compileSource(`
	func mergesort(arr) {
		length := len(arr)
		if length <= 1 {
			return arr
		}
		mid := length / 2
		left := mergesort(arr[:mid])
		right := mergesort(arr[mid:])
		output := list(length)
		i, j, k := [0, 0, 0]
		for i < len(left) {
			for j < len(right) && right[j] <= left[i] {
				output[k] = right[j]
				k++
				j++
			}
			output[k] = left[i]
			k++
			i++
		}
		for j < len(right) {
			output[k] = right[j]
			k++
			j++
		}
		return output
	}
	", ".join(mergesort([1, 9, -1, 4, 3, 2, 7, 8, 5, 6, 0]).map(string))
	`)
	require.Nil(t, err)
	data, err := MarshalCode(codeA)
	require.Nil(t, err)
	fmt.Println(string(data))
	codeB, err := UnmarshalCode(data)
	require.Nil(t, err)
	// Loops state should not factor in
	codeA.loops = nil
	for _, child := range codeA.children {
		child.loops = nil
	}
	require.Equal(t, codeA, codeB)
}

func TestSymbolTableDefinition(t *testing.T) {
	table := NewSymbolTable()
	table.InsertVariable("x")
	table.InsertConstant("c")

	def := definitionFromSymbolTable(table)
	symbols := def.Symbols
	require.Len(t, symbols, 2)

	symbol := symbols[0]
	require.Equal(t, "x", symbol.Name)
	require.Equal(t, false, symbol.IsConstant)
	require.Equal(t, uint16(0), symbol.Index)

	symbol = symbols[1]
	require.Equal(t, "c", symbol.Name)
	require.Equal(t, true, symbol.IsConstant)
	require.Equal(t, uint16(1), symbol.Index)

	newTable, err := symbolTableFromDefinition(def)
	require.Nil(t, err)
	require.Equal(t, table, newTable)
}

func TestCodeConstants(t *testing.T) {
	c := Code{symbols: NewSymbolTable()}
	c.constants = append(c.constants, int64(1), 2.0, "three", true, nil)
	data, err := MarshalCode(&c)
	require.Nil(t, err)
	c2, err := UnmarshalCode(data)
	require.Nil(t, err)
	require.Equal(t, c.constants, c2.constants)
}

func TestCompiledInstructions(t *testing.T) {
	code, err := compileSource(`1 + 2`)
	require.Nil(t, err)
	instrs := NewInstructionIter(code).All()
	require.Equal(t, [][]op.Code{
		{op.LoadConst, 0},
		{op.LoadConst, 1},
		{op.BinaryOp, op.Code(op.Add)},
	}, instrs)

	data, err := MarshalCode(code)
	require.Nil(t, err)

	code2, err := UnmarshalCode(data)
	require.Nil(t, err)

	instrs = NewInstructionIter(code2).All()
	require.Equal(t, [][]op.Code{
		{op.LoadConst, 0},
		{op.LoadConst, 1},
		{op.BinaryOp, op.Code(op.Add)},
	}, instrs)
}
