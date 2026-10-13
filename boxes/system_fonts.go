package boxes

import "github.com/robtrivi/pagecraft/fonts"

func (root *Box) loadSystemFonts() error {
	var loadTree func(*Box) error
	loadTree = func(box *Box) error {
		families := box.Style.FontFamilies
		if len(families) == 0 && box.Style.FontFamily != "" {
			families = []string{box.Style.FontFamily}
		}
		for _, family := range families {
			if _, loaded := root.FontVariants[family]; loaded {
				continue
			}
			variants, found, err := fonts.LoadSystemFamily(family)
			if err != nil {
				return err
			}
			if !found {
				continue
			}
			root.FontVariants[family] = variants
			if variants.Regular != nil {
				root.Fonts[family] = variants.Regular
			} else if variants.Bold != nil {
				root.Fonts[family] = variants.Bold
			}
			break
		}
		for _, child := range box.Children {
			if err := loadTree(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range root.Children {
		if err := loadTree(child); err != nil {
			return err
		}
	}
	return nil
}
