package content

type Content struct{
	RIndentifer string
}

func NewContent(rID string) *Content {
	return &Content{
		RIndentifer: rID,
	}
}