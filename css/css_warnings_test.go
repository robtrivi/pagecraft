package css

import (
	"reflect"
	"testing"
)

// v0.2 adds tolerant mode and warnings. Invalid declarations are still
// ignored (per CSS spec), but are reported to help debug templates.
func TestCSS_InvalidDeclaration_IsIgnoredAndWarned(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        Declarations
	}{
		{"malformed declaration", "display: block; :::; margin: 10px", Declarations{"display": "block", "margin": "10px"}},
		{"missing colon", "margin 10px; color: red", Declarations{"color": "red"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings, err := ParseInlineStyleWithWarnings(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(map[string]string(got), map[string]string(tc.want)) {
				t.Errorf("declarations: got %v, want %v", got, tc.want)
			}
			if len(warnings) != 1 {
				t.Errorf("warnings: got %v, want exactly one", warnings)
			}
		})
	}
}
