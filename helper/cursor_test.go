package helper

import (
	"testing"
	"time"

	"bansos-backend/app/model"
)

func TestCursorRoundTrip(t *testing.T) {
	in := model.Cursor{CreatedAt: time.Date(2026, 11, 25, 10, 30, 0, 123456000, time.UTC), ID: 42}

	out, err := DecodeCursor(EncodeCursor(in))
	if err != nil {
		t.Fatalf("decode gagal: %v", err)
	}
	if !out.CreatedAt.Equal(in.CreatedAt) || out.ID != in.ID {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestDecodeCursorEmpty(t *testing.T) {
	c, err := DecodeCursor("")
	if err != nil || c != nil {
		t.Fatalf("cursor kosong harus nil tanpa error, dapat %v %v", c, err)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	for _, s := range []string{"!!!", "bm9waXBl", "MjAyNi0xMS0yNXxhYmM"} {
		if _, err := DecodeCursor(s); err == nil {
			t.Errorf("cursor %q seharusnya ditolak", s)
		}
	}
}

func TestNormalizeLimit(t *testing.T) {
	cases := map[int]int{0: DefaultPageLimit, -5: DefaultPageLimit, 10: 10, 500: MaxPageLimit}
	for in, want := range cases {
		if got := NormalizeLimit(in); got != want {
			t.Errorf("NormalizeLimit(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestPaginate(t *testing.T) {
	base := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	rows := []model.Cursor{{CreatedAt: base, ID: 3}, {CreatedAt: base, ID: 2}, {CreatedAt: base, ID: 1}}
	identity := func(c model.Cursor) model.Cursor { return c }

	page, meta := Paginate(rows, 2, identity)
	if len(page) != 2 || !meta.HasMore || meta.NextCursor == "" {
		t.Fatalf("halaman pertama salah: %v %+v", page, meta)
	}

	next, _ := DecodeCursor(meta.NextCursor)
	if next.ID != 2 {
		t.Fatalf("next cursor harus menunjuk id 2, dapat %d", next.ID)
	}

	page, meta = Paginate(rows[2:], 2, identity)
	if len(page) != 1 || meta.HasMore || meta.NextCursor != "" {
		t.Fatalf("halaman terakhir salah: %v %+v", page, meta)
	}
}
