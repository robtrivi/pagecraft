package boxes

func AssignSourcePaths(root *Box) {
	if root == nil {
		return
	}
	root.Parent = nil
	root.sourcePath = nil
	var visit func(*Box)
	visit = func(parent *Box) {
		for index, child := range parent.Children {
			child.Parent = parent
			child.sourcePath = append(append([]int(nil), parent.sourcePath...), index)
			visit(child)
		}
	}
	visit(root)
}
