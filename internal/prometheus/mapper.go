package prometheus

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/alecthomas/kong"
)

var LabelFilterMapper kong.MapperFunc = func(ctx *kong.DecodeContext, target reflect.Value) error {
	var filterStr string
	if err := ctx.Scan.PopValueInto("string", &filterStr); err != nil {
		return err
	}

	kv := strings.SplitN(filterStr, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("unable to parse %q as a key-value expression", filterStr)
	}

	target.Set(reflect.ValueOf(metricsql.LabelFilter{Label: kv[0], Value: kv[1]}))
	return nil
}
