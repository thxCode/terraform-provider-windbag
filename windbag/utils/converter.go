package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ToInt tries to convert an interface to `int`.
func ToInt(i interface{}, d ...int) int {
	if i != nil {
		switch v := i.(type) {
		case int8:
			return int(v)
		case *int8:
			return int(*v)
		case int:
			return v
		case *int:
			return *v
		case int32:
			return int(v)
		case *int32:
			return int(*v)
		case int64:
			return int(v)
		case *int64:
			return int(*v)
		case uint8:
			return int(v)
		case *uint8:
			return int(*v)
		case uint:
			return int(v)
		case *uint:
			return int(*v)
		case uint32:
			return int(v)
		case *uint32:
			return int(*v)
		case uint64:
			return int(v)
		case *uint64:
			return int(*v)
		}
	}

	if len(d) != 0 {
		return d[0]
	}
	return 0
}

// ToString tries to convert an interface to `string`.
func ToString(i interface{}, d ...string) string {
	if i != nil {
		switch v := i.(type) {
		case string:
			return v
		case *string:
			return *v
		case []byte:
			return string(v)
		case *[]byte:
			return string(*v)
		}
	}

	if len(d) != 0 {
		return d[0]
	}
	return ""
}

// ToBool tries to convert an interface to `bool`.
func ToBool(i interface{}, d ...bool) bool {
	if i != nil {
		switch v := i.(type) {
		case bool:
			return v
		case *bool:
			return *v
		case string:
			var vi, err = strconv.ParseBool(v)
			if err == nil {
				return vi
			}
		case *string:
			var vi, err = strconv.ParseBool(*v)
			if err == nil {
				return vi
			}
		}
	}

	if len(d) != 0 {
		return d[0]
	}
	return false
}

// ToDuration tries to convert an interface to `time.Duration`.
func ToDuration(i interface{}, d ...time.Duration) time.Duration {
	if i != nil {
		switch v := i.(type) {
		case time.Duration:
			return v
		case *time.Duration:
			return *v
		case int64:
			return time.Duration(v)
		case *int64:
			return time.Duration(*v)
		case string:
			var vi, err = time.ParseDuration(v)
			if err == nil {
				return vi
			}
		case *string:
			var vi, err = time.ParseDuration(*v)
			if err == nil {
				return vi
			}
		}
	}

	if len(d) != 0 {
		return d[0]
	}
	return 0
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
			if v.Len() > 0 {
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
