package domain

type CommentNode struct {
	Comment
	Children []*CommentNode
}

func BuildCommentTree(comments []Comment) []*CommentNode {
	nodes := make(map[int64]*CommentNode, len(comments))
	roots := make([]*CommentNode, 0, len(comments))

	for i := range comments {
		node := &CommentNode{Comment: comments[i]}
		nodes[node.ID] = node

		if node.ParentID != nil {
			if parent, ok := nodes[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}

		roots = append(roots, node)
	}

	return roots
}
