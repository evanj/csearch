package grep

import (
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	first := FuzzyMatch("type", "a/b/c/TypeAheadHandler.java")
	secondA := FuzzyMatch("type", "a/b/c/OBJ_TYPE.txt")
	secondB := FuzzyMatch("type", "a/b/c/PlaceType.java")
	third := FuzzyMatch("type", "a/b/type/Foo.java")
	fourth := FuzzyMatch("type", "a/b/ctyped/Foo.java")

	if first <= secondA {
		t.Error("wrong order:", first, secondA)
	}
	if secondA != secondB {
		t.Error("wrong order:", secondA, secondB)
	}
	if secondB <= third {
		t.Error("wrong order:", secondB, third)
	}
	if third <= fourth {
		t.Error("wrong order:", third, fourth)
	}
	if fourth < 0 {
		t.Error("last must match:", fourth)
	}
}
