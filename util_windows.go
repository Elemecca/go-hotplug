//go:build windows

package hotplug

import (
	"slices"
)

// splitUTF16StringList converts a Windows REG_MULTI_SZ/ZZWSTR list of strings
// provided as a slice of WCHAR to a slice of strings each of which is a slice
// of WCHAR including one null termination.
func splitUTF16StringList(list []uint16) [][]uint16 {
	out := make([][]uint16, 10)
	tail := list
	for {
		nextNull := slices.Index(tail, 0)
		if nextNull <= 0 {
			break
		}

		head := tail[:nextNull+1]
		tail = tail[nextNull+1:]

		out = append(out, head)
	}

	return out
}
