package main

import (
	"errors"
	"io"
	"os"

	"github.com/itrn0/risor"
	"github.com/itrn0/risor/modules/aws"
	"github.com/itrn0/risor/modules/bcrypt"
	"github.com/itrn0/risor/modules/carbon"
	"github.com/itrn0/risor/modules/cli"
	"github.com/itrn0/risor/modules/color"
	"github.com/itrn0/risor/modules/gha"
	"github.com/itrn0/risor/modules/image"
	"github.com/itrn0/risor/modules/isatty"
	"github.com/itrn0/risor/modules/jmespath"
	k8s "github.com/itrn0/risor/modules/kubernetes"
	"github.com/itrn0/risor/modules/net"
	"github.com/itrn0/risor/modules/pgx"
	"github.com/itrn0/risor/modules/semver"
	"github.com/itrn0/risor/modules/sql"
	"github.com/itrn0/risor/modules/tablewriter"
	"github.com/itrn0/risor/modules/template"
	"github.com/itrn0/risor/modules/uuid"
	"github.com/itrn0/risor/modules/vault"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Returns a Risor option for global variable configuration.
func getGlobals() risor.Option {
	if viper.GetBool("no-default-globals") {
		return risor.WithoutDefaultGlobals()
	}

	//************************************************************************//
	// Default modules
	//************************************************************************//

	globals := map[string]any{
		"bcrypt":      bcrypt.Module(),
		"carbon":      carbon.Module(),
		"cli":         cli.Module(),
		"color":       color.Module(),
		"gha":         gha.Module(),
		"image":       image.Module(),
		"isatty":      isatty.Module(),
		"net":         net.Module(),
		"pgx":         pgx.Module(),
		"sql":         sql.Module(),
		"tablewriter": tablewriter.Module(),
		"template":    template.Module(),
		"uuid":        uuid.Module(),
	}

	//************************************************************************//
	// Modules that contribute top-level built-in functions
	//************************************************************************//

	for k, v := range jmespath.Builtins() {
		globals[k] = v
	}
	for k, v := range template.Builtins() {
		globals[k] = v
	}

	//************************************************************************//
	// Modules which are optionally present (depending on build tags).
	// If the build tag is not set then the returned module is nil.
	//************************************************************************//

	if mod := aws.Module(); mod != nil {
		globals["aws"] = mod
	}
	if mod := k8s.Module(); mod != nil {
		globals["k8s"] = mod
	}
	if mod := vault.Module(); mod != nil {
		globals["vault"] = mod
	}
	if mod := semver.Module(); mod != nil {
		globals["semver"] = mod
	}

	return risor.WithGlobals(globals)
}

func getRisorOptions() []risor.Option {
	opts := []risor.Option{
		risor.WithConcurrency(),
		risor.WithListenersAllowed(),
		getGlobals(),
	}
	if modulesDir := viper.GetString("modules"); modulesDir != "" {
		opts = append(opts, risor.WithLocalImporter(modulesDir))
	}
	return opts
}

func shouldRunRepl(cmd *cobra.Command, args []string) bool {
	if viper.GetBool("no-repl") || viper.GetBool("stdin") {
		return false
	}
	if cmd.Flags().Lookup("code").Changed {
		return false
	}
	if len(args) > 0 {
		return false
	}
	return isTerminalIO()
}

func getRisorCode(cmd *cobra.Command, args []string) (string, error) {
	// Determine what code is to be executed. There three possibilities:
	// 1. --code <code>
	// 2. --stdin (read code from stdin)
	// 3. path as args[0]
	var codeFlagSet bool
	if f := cmd.Flags().Lookup("code"); f != nil && f.Changed {
		codeFlagSet = true
	}
	var stdinFlagSet bool
	if f := cmd.Flags().Lookup("stdin"); f != nil && f.Changed {
		stdinFlagSet = true
	}
	pathSupplied := len(args) > 0
	// Error if multiple input sources are specified
	if pathSupplied && (codeFlagSet || stdinFlagSet) {
		return "", errors.New("multiple input sources specified")
	} else if codeFlagSet && stdinFlagSet {
		return "", errors.New("multiple input sources specified")
	}
	if stdinFlagSet {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(data), nil
	} else if pathSupplied {
		bytes, err := os.ReadFile(args[0])
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	return viper.GetString("code"), nil
}
