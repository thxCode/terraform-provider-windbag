package utils

// borrow from github.com/hashicorp/terraform-plugin-sdk/v2/internal/helper/hashcode

import (
	"bytes"
	"fmt"
	"hash/crc32"
)

// HashString hashes a string to a unique hashcode.
//
// crc32 returns a uint32, but for our use we need
// and non negative integer. Here we cast to an integer
// and invert it if the result is negative.
func HashString(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}

// HashStrings hashes a list of strings to a unique hashcode.
func HashStrings(strings []string) string {
	var buf bytes.Buffer
	for _, s := range strings {
		buf.WriteString(fmt.Sprintf("%s-", s))
	}
	return fmt.Sprintf("%d", HashString(buf.String()))
}
