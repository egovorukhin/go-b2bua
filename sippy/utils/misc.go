package sippy_utils

import (
	"unicode"
)

// There is no FieldsN function so here is a substitution for it
func FieldsN(s string, max_slices int) []string {
	return FieldsNFunc(s, max_slices, unicode.IsSpace)
}

func FieldsNFunc(s string, max_slices int, test_func func(rune) bool) []string {
	ret := []string{}
	buf := make([]rune, 0, len(s))
	non_space_found := false
	for _, r := range s {
		if max_slices == 0 {
			buf = append(buf, r)
			continue
		}
		if test_func(r) {
			if non_space_found {
				non_space_found = false
				ret = append(ret, string(buf))
				buf = make([]rune, 0, len(s))
			}
		} else {
			if !non_space_found {
				non_space_found = true
				if max_slices > 0 {
					max_slices--
				}
			}
			buf = append(buf, r)
		}
	}
	if len(buf) > 0 {
		ret = append(ret, string(buf))
	}
	return ret
}
