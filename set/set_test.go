package set

import "testing"

func TestAddFunc(t *testing.T) {
	// String set
	strSet := New[string]()
	if len(strSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(strSet))
	}

	Add(strSet, "x")
	if len(strSet) != 1 {
		t.Fatalf("expected set size 1, got %d", len(strSet))
	}

	// Int set
	intSet := New[int]()
	if len(intSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(intSet))
	}
	Add(intSet, 1)
	if len(intSet) != 1 {
		t.Fatalf("expected set size 1, got %d", len(intSet))
	}
}

func TestAddListFunc(t *testing.T) {
	// String set
	strSet := New[string]()
	if len(strSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(strSet))
	}

	AddList(strSet, []string{"val1", "val2"})
	if len(strSet) != 2 {
		t.Fatalf("expected set size 2, got %d", len(strSet))
	}

	// Int set
	intSet := New[int]()
	if len(intSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(intSet))
	}
	AddList(intSet, []int{1, 2})
	if len(intSet) != 2 {
		t.Fatalf("expected set size 2, got %d", len(intSet))
	}
}

func TestAddMethod(t *testing.T) {
	// String set
	strSet := New[string]()
	if len(strSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(strSet))
	}

	strSet.Add("x")
	if len(strSet) != 1 {
		t.Fatalf("expected set size 1, got %d", len(strSet))
	}

	// Int set
	intSet := New[int]()
	if len(intSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(intSet))
	}
	intSet.Add(1)
	if len(intSet) != 1 {
		t.Fatalf("expected set size 1, got %d", len(intSet))
	}
}

func TestAddListMethod(t *testing.T) {
	// String set
	strSet := New[string]()
	if len(strSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(strSet))
	}

	strSet.AddList([]string{"val1", "val2"})
	if len(strSet) != 2 {
		t.Fatalf("expected set size 2, got %d", len(strSet))
	}

	// Int set
	intSet := New[int]()
	if len(intSet) != 0 {
		t.Fatalf("expected empty set, got %d", len(intSet))
	}
	intSet.AddList([]int{1, 2})
	if len(intSet) != 2 {
		t.Fatalf("expected set size 2, got %d", len(intSet))
	}
}

func TestContainsFunc(t *testing.T) {
	strSet := New[string]()
	if Contains(strSet, "item") {
		t.Fatalf("did not expect item to exist")
	}
	strSet.Add("item")
	if !Contains(strSet, "item") {
		t.Fatalf("expected item to exist after adding")
	}

	intSet := New[int]()
	if Contains(intSet, 100) {
		t.Fatalf("did not expect 100 to exist")
	}
	intSet.Add(100)
	if !Contains(intSet, 100) {
		t.Fatalf("expected 100 to exist after adding")
	}
}

func TestContainsMethod(t *testing.T) {
	strSet := New[string]()
	if strSet.Contains("item") {
		t.Fatalf("did not expect item to exist")
	}
	strSet.Add("item")
	if !strSet.Contains("item") {
		t.Fatalf("expected item to exist after adding")
	}

	intSet := New[int]()
	if intSet.Contains(100) {
		t.Fatalf("did not expect 100 to exist")
	}
	intSet.Add(100)
	if !intSet.Contains(100) {
		t.Fatalf("expected 100 to exist after adding")
	}
}

func TestRemoveFunc(t *testing.T) {
	// String set
	strSet := New[string]()
	Add(strSet, "item1")
	Add(strSet, "item2")
	Remove(strSet, "item1")
	if Contains(strSet, "item1") {
		t.Fatalf("expected item1 to be removed")
	}
	if !Contains(strSet, "item2") {
		t.Fatalf("expected item2 to still exist")
	}

	// Int set
	intSet := New[int]()
	Add(intSet, 100)
	Add(intSet, 200)
	Remove(intSet, 100)
	if Contains(intSet, 100) {
		t.Fatalf("expected 100 to be removed")
	}
	if !Contains(intSet, 200) {
		t.Fatalf("expected 200 to still exist")
	}
}

func TestRemoveMethod(t *testing.T) {
	// String set
	strSet := New[string]()
	strSet.Add("item1")
	strSet.Add("item2")
	strSet.Remove("item1")
	if strSet.Contains("item1") {
		t.Fatalf("expected item1 to be removed")
	}
	if !strSet.Contains("item2") {
		t.Fatalf("expected item2 to still exist")
	}

	// Int set
	intSet := New[int]()
	intSet.Add(100)
	intSet.Add(200)
	intSet.Remove(100)
	if intSet.Contains(100) {
		t.Fatalf("expected 100 to be removed")
	}
	if !intSet.Contains(200) {
		t.Fatalf("expected 200 to still exist")
	}
}

func TestIsEmptyMethodClearToSlice(t *testing.T) {
	// Create a set and ensure it is initially empty
	s := New[int]()
	if !s.IsEmpty() {
		t.Fatalf("expected set to be empty right after creation")
	}

	// Add items and verify IsEmpty returns false
	s.Add(1)
	s.Add(2)
	if s.IsEmpty() {
		t.Fatalf("expected set to be non-empty after adding elements")
	}

	// Verify ToSlice contains the added elements (order not guaranteed)
	slice := s.ToSlice()
	if len(slice) != 2 {
		t.Fatalf("expected slice length 2, got %d", len(slice))
	}

	// Use Clear to empty the set and verify the result
	s.Clear()
	if !s.IsEmpty() {
		t.Fatalf("expected set to be empty after Clear")
	}
	if s.Size() != 0 {
		t.Fatalf("expected size 0 after Clear, got %d", s.Size())
	}
	if len(s.ToSlice()) != 0 {
		t.Fatalf("expected ToSlice length 0 after Clear")
	}
}
