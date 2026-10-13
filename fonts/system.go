package fonts

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	goFont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

type systemFace struct {
	path   string
	weight goFont.Weight
	italic bool
}

// SystemCatalog indexes installed TrueType fonts by their declared family.
// Font files are loaded only when a family is requested for rendering.
type SystemCatalog struct {
	mu       sync.Mutex
	families map[string][]systemFace
	loaded   map[string]*Face
}

// NewSystemCatalog scans font directories in deterministic path order.
func NewSystemCatalog(dirs ...string) *SystemCatalog {
	catalog := &SystemCatalog{
		families: make(map[string][]systemFace),
		loaded:   make(map[string]*Face),
	}
	var paths []string
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() || !isTrueTypePath(path) {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
	}
	sort.Strings(paths)
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		loader, err := opentype.NewLoader(file)
		if err != nil {
			file.Close()
			continue
		}
		description, _ := goFont.Describe(loader, nil)
		aspect := description.Aspect
		file.Close()
		family := normalizeFamily(description.Family)
		if family == "" {
			continue
		}
		catalog.families[family] = append(catalog.families[family], systemFace{
			path: path, weight: aspect.Weight, italic: aspect.Style == goFont.StyleItalic,
		})
	}
	return catalog
}

// LoadFamily loads the closest regular, bold, italic, and bold-italic faces.
func (c *SystemCatalog) LoadFamily(family string) (Variants, bool, error) {
	if c == nil {
		return Variants{}, false, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	candidates := c.familyCandidates(family)
	for _, candidate := range candidates {
		faces := c.families[normalizeFamily(candidate)]
		if len(faces) == 0 {
			continue
		}
		var variants Variants
		var err error
		variants.Regular, err = c.loadClosest(faces, false, goFont.WeightNormal)
		if err != nil {
			return Variants{}, false, err
		}
		variants.Bold, err = c.loadClosest(faces, false, goFont.WeightBold)
		if err != nil {
			return Variants{}, false, err
		}
		variants.Italic, err = c.loadClosest(faces, true, goFont.WeightNormal)
		if err != nil {
			return Variants{}, false, err
		}
		variants.BoldItalic, err = c.loadClosest(faces, true, goFont.WeightBold)
		if err != nil {
			return Variants{}, false, err
		}
		return variants, true, nil
	}
	return Variants{}, false, nil
}

func (c *SystemCatalog) loadClosest(faces []systemFace, italic bool, weight goFont.Weight) (*Face, error) {
	var best *systemFace
	for i := range faces {
		face := &faces[i]
		if face.italic != italic {
			continue
		}
		if best == nil || absWeight(face.weight-weight) < absWeight(best.weight-weight) {
			best = face
		}
	}
	if best == nil {
		return nil, nil
	}
	if face := c.loaded[best.path]; face != nil {
		return face, nil
	}
	data, err := os.ReadFile(best.path)
	if err != nil {
		return nil, err
	}
	face, err := Load(data)
	if err != nil {
		return nil, err
	}
	c.loaded[best.path] = face
	return face, nil
}

func (c *SystemCatalog) familyCandidates(family string) []string {
	name := normalizeFamily(family)
	if name != "sansserif" && name != "serif" && name != "monospace" {
		return []string{family}
	}
	aliases := map[string][]string{
		"sansserif": {"Arial", "Liberation Sans", "DejaVu Sans", "Noto Sans"},
		"serif":     {"Times New Roman", "Liberation Serif", "DejaVu Serif", "Noto Serif"},
		"monospace": {"Courier New", "Liberation Mono", "DejaVu Sans Mono", "Noto Sans Mono"},
	}
	return aliases[name]
}

var defaultSystemCatalog struct {
	sync.Once
	catalog *SystemCatalog
}

// LoadSystemFamily resolves an installed family, with CSS generic-family fallback.
func LoadSystemFamily(family string) (Variants, bool, error) {
	defaultSystemCatalog.Do(func() {
		defaultSystemCatalog.catalog = NewSystemCatalog(defaultFontDirectories()...)
	})
	return defaultSystemCatalog.catalog.LoadFamily(family)
}

func defaultFontDirectories() []string {
	var dirs []string
	add := func(paths ...string) {
		for _, path := range paths {
			if path != "" {
				dirs = append(dirs, path)
			}
		}
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		add(os.Getenv("WINDIR")+`\Fonts`, filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts"))
	case "darwin":
		add("/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library", "Fonts"))
	default:
		add("/usr/share/fonts", "/usr/local/share/fonts", filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts"))
	}
	return dirs
}

func isTrueTypePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ttf" || ext == ".otf"
}

func normalizeFamily(family string) string {
	return goFont.NormalizeFamily(strings.TrimSpace(family))
}

func absWeight(weight goFont.Weight) goFont.Weight {
	if weight < 0 {
		return -weight
	}
	return weight
}
