package css

import (
	"reflect"
	"testing"
)

// Stylesheet step 1: selector rules and declarations; at-rules are ignored
// in tolerant mode. Cascade is covered in boxes/.
func TestCSS_ParseStylesheet_ReturnsRulesInSourceOrder(t *testing.T) {
	cases := []struct {
		name, input string
		want        []Rule
	}{
		{
			name:  "style rules with ignored at-rule",
			input: "p { color: red; } .a { margin: 2px; } @media x { p { color: blue; } }",
			want: []Rule{
				{Selector: "p", Decls: Declarations{"color": "red"}, Order: 0},
				{Selector: ".a", Decls: Declarations{"margin": "2px"}, Order: 1},
			},
		},
		{name: "empty stylesheet", input: "", want: nil},
		{
			name:  "rule after at-rule",
			input: `@page { size: A4; } p { color: red; }`,
			want:  []Rule{{Selector: "p", Decls: Declarations{"color": "red"}, Order: 0}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseStylesheet(tc.input)
			if err != nil {
				t.Fatalf("ParseStylesheet: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("rules: got %+v, want %+v", got, tc.want)
			}
		})
	}
}
