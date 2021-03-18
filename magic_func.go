package httpwrapper

import (
	"math/rand"
	"reflect"
	"time"

	"github.com/hpgood/boomer"
	"github.com/spf13/cast"
)

var TemplateFunc = map[string]interface{}{
	"getRandomId": getRandomId,
	"getSid":      getSid,
	"toFloat64":   cast.ToFloat64,
	"toString":    cast.ToString,
	"mapValue":    MapValue,
	"storeValue":  StoreValue,
}

// StoreValue 
func StoreValue(ctx * boomer.RunContext,key string) string {
	if ctx==nil{
		return NoValue
	}
	if v,ok:=ctx.Store[key];ok {
		return v
	}
	return ""
}
// MapValue
func MapValue(m interface{},k string) interface{} {
	
	// fmt.Println("@MapValue k=",k)

	v:=reflect.TypeOf(m)
	switch(v.Kind()){
		case reflect.String:
			return ""
		case reflect.Map:
			var v2 map[string] interface{}= m.(map[string] interface{})
			if value,ok:=v2[k];ok {
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
