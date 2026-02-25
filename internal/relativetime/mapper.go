package relativetime

import (
	"reflect"

	"github.com/alecthomas/kong"
)

var Mapper kong.MapperFunc = func(ctx *kong.DecodeContext, target reflect.Value) error {
	var timeStr string
	ctx.Scan.PopValueInto("string", &timeStr)
	time, err := Parse(timeStr)
	if err != nil {
		return err
	}
	target.Set(reflect.ValueOf(time))
	return nil
}
