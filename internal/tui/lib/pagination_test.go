package lib

import "testing"

func TestPaginateSplitsByWindowHeight(t *testing.T) {
	t.Parallel()

	page := Paginate(7, 3, 7, 4)
	if page.Start != 3 || page.End != 6 {
		t.Fatalf("expected second page 3-6, got %d-%d", page.Start, page.End)
	}
	if page.PageSize != 3 {
		t.Fatalf("expected page size 3, got %d", page.PageSize)
	}
	if label := page.StatusLabel(7); label != "4-6/7" {
		t.Fatalf("expected status label 4-6/7, got %q", label)
	}
}

func TestPaginateFallsBackForUnknownWindowHeight(t *testing.T) {
	t.Parallel()

	page := Paginate(30, 0, 0, 4)
	if page.PageSize != 20 {
		t.Fatalf("expected default page size 20, got %d", page.PageSize)
	}
	if page.Start != 0 || page.End != 20 {
		t.Fatalf("expected first page 0-20, got %d-%d", page.Start, page.End)
	}
}
