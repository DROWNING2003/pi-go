package session

// TreeNode represents a node in the session tree (fork/clone hierarchy).
type TreeNode struct {
	ID       string
	ParentID string
	Label    string
	Children []*TreeNode
}

// Tree builds a tree from a list of session info entries.
func BuildTree(sessions []Info) []*TreeNode {
	nodeMap := make(map[string]*TreeNode)
	var roots []*TreeNode

	// Create nodes
	for _, s := range sessions {
		nodeMap[s.ID] = &TreeNode{ID: s.ID}
	}

	// Build parent-child relationships using the header's parentSession
	for _, s := range sessions {
		sess, err := Load(s.Path)
		if err != nil {
			continue
		}
		node := nodeMap[s.ID]
		parentID := sess.Header.ParentSession

		if parentID != "" {
			if parent, ok := nodeMap[parentID]; ok {
				node.ParentID = parentID
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	return roots
}

// Walk traverses the tree depth-first, calling fn for each node.
func Walk(roots []*TreeNode, fn func(*TreeNode, int)) {
	for _, root := range roots {
		walkNode(root, 0, fn)
	}
}

func walkNode(node *TreeNode, depth int, fn func(*TreeNode, int)) {
	fn(node, depth)
	for _, child := range node.Children {
		walkNode(child, depth+1, fn)
	}
}

// FindNode finds a node by ID in the tree.
func FindNode(roots []*TreeNode, id string) *TreeNode {
	for _, root := range roots {
		if n := findNode(root, id); n != nil {
			return n
		}
	}
	return nil
}

func findNode(node *TreeNode, id string) *TreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if n := findNode(child, id); n != nil {
			return n
		}
	}
	return nil
}

// Path returns the path of nodes from root to the given node.
func Path(roots []*TreeNode, id string) []*TreeNode {
	node := FindNode(roots, id)
	if node == nil {
		return nil
	}
	// Build path from root to node
	var path []*TreeNode
	buildPath(roots, node, &path)
	return path
}

func buildPath(roots []*TreeNode, target *TreeNode, path *[]*TreeNode) bool {
	for _, root := range roots {
		if buildPathNode(root, target, path) {
			return true
		}
	}
	return false
}

func buildPathNode(node, target *TreeNode, path *[]*TreeNode) bool {
	*path = append(*path, node)
	if node == target {
		return true
	}
	for _, child := range node.Children {
		if buildPathNode(child, target, path) {
			return true
		}
	}
	*path = (*path)[:len(*path)-1]
	return false
}
