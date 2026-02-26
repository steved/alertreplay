package relativetime

import (
	"reflect"

	"github.com/alecthomas/kong"
)

var Mapper kong.MapperFunc = func(ctx *kong.DecodeContext, target reflect.Value) error {
	var timeStr string

	if err := ctx.Scan.PopValueInto("string", &timeStr); err != nil {
		return err
	}

	time, err := Parse(timeStr)
	if err != nil {
		return err
	}

	target.Set(reflect.ValueOf(time))
	return nil
}
