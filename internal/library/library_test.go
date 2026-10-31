package library

import (
	"errors"
	"testing"
)

func TestValidName(t *testing.T) {
	for _, ok := range []string{"Gig 1", "a", "intro_v2.final", "Set (B)"} {
		if err := ValidName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", ".hidden", "../etc/passwd", "a/b", "a..b", " lead", "x\\y"} {
		if err := ValidName(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Write(Shows, "Gig 1", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List(Shows)
	if len(list) != 1 || list[0].Name != "Gig 1" {
		t.Fatalf("list = %+v", list)
	}
	b, err := s.Read(Shows, "Gig 1")
	if err != nil || string(b) != `{"a":1}` {
		t.Fatalf("read = %s, %v", b, err)
	}
	if err := s.Delete(Shows, "Gig 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(Shows, "Gig 1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("read after delete: %v", err)
	}
	if err := s.Write("bogus", "x", nil); err == nil {
		t.Error("expected error for bad kind")
	}
}
