package css

import (
	"reflect"
	"testing"
)

// Parse inline declarations in the
// initial subset (display, margin, font-*, color). Cascade is not covered yet.
func TestCSS_ParsesDisplayMargin(t *testing.T) {
	got, err := ParseInlineStyle("display: block; margin: 10px; color: red; font-size: 16px")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Declarations{
		"display":   "block",
		"margin":    "10px",
		"color":     "red",
		"font-size": "16px",
	}
	if !reflect.DeepEqual(map[string]string(got), map[string]string(want)) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCSS_ParsesFlexShorthandAsOneDeclaration(t *testing.T) {
	got, err := ParseInlineStyle("flex: 1 1 0%; flex-basis: 25%")
	if err != nil {
		t.Fatalf("ParseInlineStyle: %v", err)
	}
	if got["flex"] != "1 1 0%" || got["flex-basis"] != "25%" {
		t.Errorf("flex declarations = %#v, want full shorthand and basis values", got)
	}
}

func TestCSS_ParsesInlineStyle_EdgeCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Declarations
	}{
		{name: "empty", in: "", want: Declarations{}},
		{name: "without spaces", in: "color:red", want: Declarations{"color": "red"}},
		{name: "uppercase property", in: "Margin: 5px", want: Declarations{"margin": "5px"}},
		{name: "important", in: "margin: 10px !important", want: Declarations{"margin": "10px"}},
		{name: "invalid text is ignored", in: ":::; display: block", want: Declarations{"display": "block"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseInlineStyle(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(map[string]string(got), map[string]string(tc.want)) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
