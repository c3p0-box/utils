package set

import "testing"

func TestAdd(t *testing.T) {
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

func TestContains(t *testing.T) {
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

func TestRemove(t *testing.T) {
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

func TestIsEmptyClearToSlice(t *testing.T) {
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
