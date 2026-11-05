package jsonview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFormatPlain(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"scalar", `42`, `42`},
		{"empty containers", `{"a":{},"b":[]}`, "{\n  \"a\": {},\n  \"b\": []\n}"},
		{"order kept", `{"z":1,"a":[true,null,"x"]}`, "{\n  \"z\": 1,\n  \"a\": [\n    true,\n    null,\n    \"x\"\n  ]\n}"},
		{"big number", `12345678901234567890`, `12345678901234567890`},
		{"no html escape", `"<a&b>"`, `"<a&b>"`},
		{"unicode", `"héllo 日本"`, `"héllo 日本"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format([]byte(tt.in), Options{})
			if err != nil || got != tt.want {
				t.Errorf("got %q (%v), want %q", got, err, tt.want)
			}
		})
	}
}

func TestFormatInvalid(t *testing.T) {
	for _, in := range []string{``, `{`, `{"a":}`, `1 2`, `nope`} {
		if _, err := Format([]byte(in), Options{}); err == nil {
			t.Errorf("Format(%q) should fail", in)
		}
	}
}

func TestFormatColors(t *testing.T) {
	const in = `{"k":"v","n":1,"b":true,"z":null}`
	got, err := Format([]byte(in), Options{Color: true})
	if err != nil {
		t.Fatal(err)
	}
	if ansi.Strip(got) == got {
		t.Fatal("expected ANSI colors")
	}
	for _, want := range []string{"36m", "32m", "33m", "35m"} { // cyan green yellow magenta
		if !strings.Contains(got, want) {
			t.Errorf("missing color %s in %q", want, got)
		}
	}
	plain, _ := Format([]byte(in), Options{})
	if ansi.Strip(got) != plain {
		t.Errorf("stripped colored output differs from plain:\n%s\n%s", ansi.Strip(got), plain)
	}
}

func TestTruncateLongStrings(t *testing.T) {
	long := strings.Repeat("A", 500)
	in := `{"img":"` + long + `","short":"ok"}`
	got, _ := Format([]byte(in), Options{MaxString: DefaultMaxString})
	if strings.Contains(got, long) || !strings.Contains(got, "… (500 chars)") || !strings.Contains(got, `"ok"`) {
		t.Errorf("unexpected: %s", got)
	}
	full, _ := Format([]byte(in), Options{})
	if !strings.Contains(full, long) {
		t.Error("full mode must keep the whole string")
	}
	// exactly at the limit is not truncated
	edge, _ := Format([]byte(`"`+strings.Repeat("x", 200)+`"`), Options{MaxString: 200})
	if strings.Contains(edge, "chars)") {
		t.Error("string of exactly MaxString runes must not be truncated")
	}
	// rune count, not bytes
	uni, _ := Format([]byte(`"`+strings.Repeat("日", 201)+`"`), Options{MaxString: 200})
	if !strings.Contains(uni, "(201 chars)") {
		t.Errorf("got %q", uni)
	}
}

func TestDeepAndHuge(t *testing.T) {
	deep := strings.Repeat("[", MaxDepth+10) + strings.Repeat("]", MaxDepth+10)
	if _, err := Format([]byte(deep), Options{}); err == nil {
		t.Error("too deep input must be rejected, not crash")
	}
	ok := strings.Repeat("[", 100) + strings.Repeat("]", 100)
	if _, err := Format([]byte(ok), Options{}); err != nil {
		t.Errorf("depth 100 should work: %v", err)
	}
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < 50000; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"id":1,"name":"item"}`)
	}
	b.WriteString("]")
	out, err := Format([]byte(b.String()), Options{Color: true})
	if err != nil || strings.Count(out, "\n") < 50000 {
		t.Errorf("huge array failed: %v", err)
	}
}

func TestBody(t *testing.T) {
	o := Options{MaxString: 200}
	if _, ok := Body(nil, o); ok {
		t.Error("absent must report !ok")
	}
	if _, ok := Body([]byte(`null`), o); ok {
		t.Error("null must report !ok")
	}
	if s, _ := Body([]byte(`"Not Found"`), o); s != "Not Found" {
		t.Errorf("plain string = %q", s)
	}
	if s, _ := Body([]byte(`"{\"a\":1}"`), o); s != "{\n  \"a\": 1\n}" {
		t.Errorf("json-in-string = %q", s)
	}
	if s, _ := Body([]byte(`"{broken"`), o); s != "{broken" {
		t.Errorf("broken json-in-string = %q", s)
	}
	if s, _ := Body([]byte(`<html>`), o); s != "<html>" {
		t.Errorf("raw non-json = %q", s)
	}
	long := strings.Repeat("z", 300)
	if s, _ := Body([]byte(`"`+long+`"`), o); !strings.Contains(s, "(300 chars)") {
		t.Errorf("long text not truncated: %q", s)
	}
}
