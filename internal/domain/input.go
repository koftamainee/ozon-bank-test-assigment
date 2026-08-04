package domain

type LoginInput struct {
	Username Username
}

type CreatePostInput struct {
	Title string
	Body  string
}

type CreateCommentInput struct {
	PostID   int64
	ParentID *int64
	Body     CommentBody
}
