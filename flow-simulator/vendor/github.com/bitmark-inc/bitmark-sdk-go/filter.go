package bitmarksdk

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

type BitmarkFilter struct {
	AssetId   string `url:"asset_id"`
	Issuer    string `url:"issuer"`
	Owner     string `url:"owner"`
	OwnerSent bool   `url:"owner_sent"`
	Asset     bool   `url:"asset"`
	Pending   bool   `url:"pending"`
	To        string `url:"to"`
	At        uint   `url:"at"`
	Limit     uint   `url:"limit" validate:"max=100"`
}

type AssetFilter struct {
	Registrant string   `url:"registrant"`
	AssetIds   []string `url:"asset_ids"`
	Pending    bool     `url:"pending"`
}

func toURLValues(i interface{}) (values url.Values) {
	values = url.Values{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		f := iVal.Field(i)
		var v string
		switch f.Interface().(type) {
		case string:
			v = f.String()
			if v == "" {
				continue
			}
		case bool:
			v = strconv.FormatBool(f.Bool())
		case uint:
			if f.Uint() == 0 {
				continue
			}

			v = strconv.FormatUint(f.Uint(), 10)

			var max uint
			n, _ := fmt.Sscanf(typ.Field(i).Tag.Get("validate"), "max=%d", &max)
			if n > 0 {
				if uint(f.Uint()) > max {
					v = strconv.FormatUint(uint64(max), 10)
				}
			}
		}
		values.Set(typ.Field(i).Tag.Get("url"), v)
	}
	return
}
