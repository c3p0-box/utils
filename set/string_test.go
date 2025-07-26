package set

import (
	"strings"
	"testing"
)

func TestFromStr(t *testing.T) {
	str := "a b c  -"
	sep := " "
	items := FromStr(str, sep)

	if items.Size() != 3 {
		t.Fatalf("expected 3 items, got %d", items.Size())
	}

	for _, v := range []string{"a", "b", "c"} {
		if !items.Contains(v) {
			t.Fatalf("expected set to contain %s", v)
		}
	}
}

func TestToStr(t *testing.T) {
	items := New[string]()
	for _, v := range []string{"a", "b", "c"} {
		items.Add(v)
	}

	result := ToStr(items, ",")

	if len(result) != 5 { // e.g., "a,b,c" is 5 chars long
		t.Fatalf("expected result length 5, got %d", len(result))
	}

	for _, v := range []string{"a", "b", "c"} {
		if !strings.Contains(result, v) {
			t.Fatalf("expected result to contain %s", v)
		}
	}

	if result[1] != ',' || result[3] != ',' {
		t.Fatalf("expected commas at positions 1 and 3, got %q and %q", result[1], result[3])
	}
}
