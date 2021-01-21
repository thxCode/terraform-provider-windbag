package utils

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ToInt tries to convert an interface to `int`.
func ToInt(i interface{}) int {
	if i == nil {
		return 0
	}

	switch v := i.(type) {
	case int8:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	}
	var vi, _ = strconv.ParseInt(fmt.Sprint(i), 10, 64)
	return int(vi)
}

// ToString tries to convert an interface to `string`.
func ToString(i interface{}) string {
	if i == nil {
		return ""
	}

	switch v := i.(type) {
	case string:
		return v
	}
	return fmt.Sprint(i)
}

// ToBool tries to convert an interface to `bool`.
func ToBool(i interface{}) bool {
	if i == nil {
		return false
	}

	switch v := i.(type) {
	case bool:
		return v
	}
	var vi, _ = strconv.ParseBool(fmt.Sprint(i))
	return vi
}

// ToInterfaceSlice tries to convert an interface to `[]interface`.
func ToInterfaceSlice(i interface{}) []interface{} {
	if i != nil {
		switch v := i.(type) {
		case []interface{}:
			return v
		case *schema.Set:
			return v.List()
		}
	}
	return []interface{}{}
}

// ToStringSlice tries to convert an interface to `[]string`.
func ToStringSlice(i interface{}) []string {
	if i != nil {
		switch v := i.(type) {
		case []string:
			return v
		case []interface{}:
			var ret = make([]string, 0, len(v))
			for _, vi := range v {
				switch str := vi.(type) {
				case string:
					ret = append(ret, str)
				default:
					ret = append(ret, fmt.Sprint(vi))
				}
			}
			return ret
		}
	}
	return []string{}
}

// ToStringInterfaceMap tries to convert an interface to `map[string]interface{}`.
func ToStringInterfaceMap(i interface{}) map[string]interface{} {
	if i != nil {
		switch v := i.(type) {
		case map[string]interface{}:
			return v
		case *schema.Set:
			if v.Len() == 1 {
				return v.List()[0].(map[string]interface{})
			}
		}
	}
	return map[string]interface{}{}
}

// ToStringStringMap tries to convert an interface to `map[string]string`.
func ToStringStringMap(i interface{}) map[string]string {
	if i != nil {
		switch v := i.(type) {
		case map[string]string:
			return v
		}
	}
	var ret = map[string]string{}
	for k, v := range ToStringInterfaceMap(i) {
		ret[k] = ToString(v)
	}
	return ret
}
