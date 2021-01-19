package utils

import (
	"fmt"
	"strconv"
)

// ToInt tries to convert an interface to int value.
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

// ToString tries to convert an interface to string value.
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

// ToBool tries to convert an interface to bool value.
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

// ToStringSlice tries to convert an interface to string slice.
func ToStringSlice(i interface{}) []string {
	if i == nil {
		return nil
	}

	switch v := i.(type) {
	case []string:
		return v
	case []interface{}:
		var strs = make([]string, 0, len(v))
		for _, vi := range v {
			switch str := vi.(type) {
			case string:
				strs = append(strs, str)
			default:
				strs = append(strs, fmt.Sprint(vi))
			}
		}
		return strs
	}
	return []string{}
}
