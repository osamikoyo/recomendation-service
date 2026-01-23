package content

type Content struct{
	LastID string
	RIndentifer string
	Actions int
}

func NewContent(lastID, rID string) *Content {
	return &Content{
		LastID: lastID,
		RIndentifer: rID,
		Actions: 0,
	}
}