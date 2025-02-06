package main

import (
	"context"
	"fmt"
	"os"

	"github.com/itrn0/risor"
	"github.com/itrn0/risor/compiler"
	"github.com/itrn0/risor/dis"
	"github.com/itrn0/risor/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const disExample = `  risor dis -c "a := 1 + 2"

  risor dis ./path/to/script.risor

  risor dis ./path/to/script.risor --func myfunc`

var disCmd = &cobra.Command{
	Use:     "dis",
	Short:   "Disassemble Risor code",
	Example: disExample,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		processGlobalFlags()
		opts := getRisorOptions()
		code, err := getRisorCode(cmd, args)
		if err != nil {
			fatal(err)
		}

		// Parse then compile the input code
		ast, err := parser.Parse(ctx, code)
		if err != nil {
			fatal(err)
		}
		cfg := risor.NewConfig(opts...)
		compiledCode, err := compiler.Compile(ast, cfg.CompilerOpts()...)
		if err != nil {
			fatal(err)
		}
		targetCode := compiledCode

		// If a function name was provided, disassemble its code only
		if funcName := viper.GetString("func"); funcName != "" {
			var fn *compiler.Function
			for i := 0; i < compiledCode.ConstantsCount(); i++ {
				obj, ok := compiledCode.Constant(i).(*compiler.Function)
				if !ok {
					continue
				}
				if obj.Name() == funcName {
					fn = obj
					break
				}
			}
			if fn == nil {
				fatal(fmt.Sprintf("function %q not found", funcName))
			}
			targetCode = fn.Code()
		}

		// Disassemble and print the instructions
		instructions, err := dis.Disassemble(targetCode)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		dis.Print(instructions, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(disCmd)
	disCmd.Flags().String("func", "", "Function name")
	viper.BindPFlag("func", disCmd.Flags().Lookup("func"))
}
