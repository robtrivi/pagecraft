package css

import (
	"reflect"
	"testing"
)

// Font-face extraction shares one output contract across optional/default and
// explicitly declared descriptors. Unrelated rules do not become font faces.
func TestCSS_ParseFontFacesExtractsSupportedDescriptors(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        []FontFace
	}{
		{
			name: "default descriptors and unrelated rule",
			input: `@font-face { font-family: "Mi Tipo"; src: url(a.ttf); }
				p { color: red; }
				@font-face { font-family: Otra; src: url(data:font/ttf;base64,AAA); }`,
			want: []FontFace{
				{Family: "Mi Tipo", Src: "a.ttf", Weight: 400, Style: "normal"},
				{Family: "Otra", Src: "data:font/ttf;base64,AAA", Weight: 400, Style: "normal"},
			},
		},
		{
			name:  "explicit weight and style",
			input: `@font-face { font-family: Compare; src: url(regular.ttf); font-weight: 400; font-style: normal; } @font-face { font-family: Compare; src: url(bold-italic.ttf); font-weight: 700; font-style: italic; }`,
			want: []FontFace{
				{Family: "Compare", Src: "regular.ttf", Weight: 400, Style: "normal"},
				{Family: "Compare", Src: "bold-italic.ttf", Weight: 700, Style: "italic"},
			},
		},
		{name: "stylesheet without font faces", input: `p { color: red; }`, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFontFaces(tc.input)
			if err != nil {
				t.Fatalf("ParseFontFaces: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("font faces: got %+v, want %+v", got, tc.want)
			}
		})
	}
}
