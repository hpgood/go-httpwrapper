package httpwrapper

import (
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/hpgood/boomer"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

var TemplateFunc = map[string]interface{}{
	"getRandomId": getRandomId,
	"getSid":      getSid,
	"toFloat64":   cast.ToFloat64,
	"toString":    cast.ToString,
	"mapValue":    MapValue,
	"storeValue":  StoreValue,
	"sv":          StoreValue,
	"gson":        Gson,
	"gsonArray":   GsonStringArr,
	"joins":       JoinS,
	"join":        Join,
	"sleep":       Sleep,
}

func Sleep(n int) string {
	if n > 0 {
		time.Sleep(time.Millisecond * time.Duration(n))
	}
	return ""
}
func Gson(ctx *boomer.RunContext, p string) string {
	return gjson.Get(ctx.RspJSON, p).String()
}

// func GsonResult(ctx *boomer.RunContext, p string) gjson.Result {
// 	return gjson.Get(ctx.RspJSON, p)
// }
func GsonStringArr(ctx *boomer.RunContext, p string) []string {
	arr := gjson.Get(ctx.RspJSON, p).Array()
	ret := []string{}
	for _, v := range arr {
		ret = append(ret, v.String())
	}
	return ret
}
func JoinS(arr []string) string {
	return strings.Join(arr, ",")
}
func Join(arr []string, s string) string {
	return strings.Join(arr, s)
}

// StoreValue
func StoreValue(ctx *boomer.RunContext, key string) string {
	if ctx == nil {
		return NoValue
	}
	if v, ok := ctx.Store[key]; ok {
		return v
	}
	return ""
}

// MapValue
func MapValue(m interface{}, k string) interface{} {

	// fmt.Println("@MapValue k=",k)

	v := reflect.TypeOf(m)
	switch v.Kind() {
	case reflect.String:
		return ""
	case reflect.Map:
		var v2 map[string]interface{} = m.(map[string]interface{})
		if value, ok := v2[k]; ok {
			return value
		}
		return ""
	}
	return ""
}
func getSid() int64 {
	return time.Now().Unix()
}

func getRandomId(id int) int {
	return rand.Intn(id)
}
