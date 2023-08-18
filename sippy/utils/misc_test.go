package sippy_utils

import (
	"testing"
)

func TestFuncN(t *testing.T) {
	arr := FieldsN("  foo bar  baz   ", 2)
	if len(arr) != 2 {
		t.Errorf("String splitted into %d parts (want 2)", len(arr))
		return
	}
	if arr[0] != "foo" {
		t.Errorf("Bad first part '%s' (want 'foo')", arr[0])
	}
	if arr[1] != "bar  baz   " {
		t.Errorf("Bad first part '%s' (want 'bar  baz   ')", arr[0])
	}
}
