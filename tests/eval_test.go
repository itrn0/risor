package tests

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/itrn0/risor"
	"github.com/itrn0/risor/object"
	"github.com/itrn0/risor/parser"
	"github.com/stretchr/testify/require"
)

type TestCase struct {
	Name              string
	Text              string
	ExpectedValue     string
	ExpectedType      string
	ExpectedErr       string
	ExpectedErrLine   int
	ExpectedErrColumn int
}

func readFile(name string) string {
	data, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func parseExpectedValue(filename, text string) (TestCase, error) {
	result := TestCase{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "// ") {
			continue
		}
		line = strings.TrimPrefix(line, "// ")
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "expected value":
			result.ExpectedValue = val
		case "expected type":
			result.ExpectedType = val
		case "expected error":
			result.ExpectedErr = val
		case "expected error line":
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return result, err
			}
			result.ExpectedErrLine = intVal
		case "expected error column":
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return result, err
			}
			result.ExpectedErrColumn = intVal
		}
	}
	return result, nil
}

func getTestCase(name string) (TestCase, error) {
	input := readFile(name)
	testCase, err := parseExpectedValue(name, input)
	testCase.Name = name
	testCase.Text = input
	return testCase, err
}

func execute(ctx context.Context, input string) (object.Object, error) {
	return risor.Eval(ctx, input)
}

func listTestFiles() []string {
	files, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}
	var testFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".tm") {
			testFiles = append(testFiles, f.Name())
		}
	}
	return testFiles
}

func TestFiles(t *testing.T) {
	only := "" // test-2022-12-03-08-12
	for _, name := range listTestFiles() {
		if !strings.HasSuffix(name, ".tm") {
			continue
		}
		if only != "" && !strings.Contains(name, only) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			tc, err := getTestCase(name)
			require.Nil(t, err)
			ctx := context.Background()
			result, err := execute(ctx, tc.Text)
			expectedType := object.Type(tc.ExpectedType)

			if tc.ExpectedValue != "" {
				if result == nil {
					t.Fatalf("expected value %q, got nil", tc.ExpectedValue)
				} else {
					require.Equal(t, tc.ExpectedValue, result.Inspect())
				}
			}
			if tc.ExpectedType != "" {
				if result == nil {
					t.Fatalf("expected type %q, got nil", tc.ExpectedType)
				} else {
					require.Equal(t, expectedType, result.Type())
				}
			}
			if tc.ExpectedErr != "" {
				require.NotNil(t, err)
				require.Equal(t, tc.ExpectedErr, err.Error())
			}
			if tc.ExpectedErrColumn != 0 {
				require.NotNil(t, err)
				parserErr, ok := err.(parser.ParserError)
				require.True(t, ok)
				fmt.Println("--- Friendly error output for", name)
				fmt.Println(parserErr.FriendlyErrorMessage())
				fmt.Println("---")
				require.Equal(t,
					tc.ExpectedErrColumn,
					parserErr.StartPosition().ColumnNumber(),
					"The column number is incorrect")
			}
			if tc.ExpectedErrLine != 0 {
				require.NotNil(t, err)
				parserErr, ok := err.(parser.ParserError)
				require.True(t, ok)
				fmt.Println("--- Friendly error output for", name)
				fmt.Println(parserErr.FriendlyErrorMessage())
				fmt.Println("---")
				require.Equal(t,
					tc.ExpectedErrLine,
					parserErr.StartPosition().LineNumber(),
					"The line number is incorrect")
			}
		})
	}
}
