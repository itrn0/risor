package all

import (
	"github.com/itrn0/risor/builtins"
	modBase64 "github.com/itrn0/risor/modules/base64"
	modBytes "github.com/itrn0/risor/modules/bytes"
	modColor "github.com/itrn0/risor/modules/color"
	modErrors "github.com/itrn0/risor/modules/errors"
	modExec "github.com/itrn0/risor/modules/exec"
	modFilepath "github.com/itrn0/risor/modules/filepath"
	modFmt "github.com/itrn0/risor/modules/fmt"
	modGha "github.com/itrn0/risor/modules/gha"
	modHTTP "github.com/itrn0/risor/modules/http"
	modIsTTY "github.com/itrn0/risor/modules/isatty"
	modJSON "github.com/itrn0/risor/modules/json"
	modMath "github.com/itrn0/risor/modules/math"
	modNet "github.com/itrn0/risor/modules/net"
	modOs "github.com/itrn0/risor/modules/os"
	modRand "github.com/itrn0/risor/modules/rand"
	modRegexp "github.com/itrn0/risor/modules/regexp"
	modStrconv "github.com/itrn0/risor/modules/strconv"
	modStrings "github.com/itrn0/risor/modules/strings"
	modTablewriter "github.com/itrn0/risor/modules/tablewriter"
	modTime "github.com/itrn0/risor/modules/time"
	modYAML "github.com/itrn0/risor/modules/yaml"
	"github.com/itrn0/risor/object"
)

func Builtins() map[string]object.Object {
	result := map[string]object.Object{
		"base64":      modBase64.Module(),
		"bytes":       modBytes.Module(),
		"color":       modColor.Module(),
		"errors":      modErrors.Module(),
		"exec":        modExec.Module(),
		"filepath":    modFilepath.Module(),
		"fmt":         modFmt.Module(),
		"gha":         modGha.Module(),
		"http":        modHTTP.Module(),
		"isatty":      modIsTTY.Module(),
		"json":        modJSON.Module(),
		"math":        modMath.Module(),
		"net":         modNet.Module(),
		"os":          modOs.Module(),
		"rand":        modRand.Module(),
		"regexp":      modRegexp.Module(),
		"strconv":     modStrconv.Module(),
		"strings":     modStrings.Module(),
		"tablewriter": modTablewriter.Module(),
		"time":        modTime.Module(),
		"yaml":        modYAML.Module(),
	}
	for k, v := range modHTTP.Builtins() {
		result[k] = v
	}
	for k, v := range modFmt.Builtins() {
		result[k] = v
	}
	for k, v := range builtins.Builtins() {
		result[k] = v
	}
	for k, v := range modOs.Builtins() {
		result[k] = v
	}
	return result
}
