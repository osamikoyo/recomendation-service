package core

type rec struct {
	Value int
	Rid   string
}

type Recs []rec

func (a Recs) Len() int           { return len(a) }
func (a Recs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Recs) Less(i, j int) bool { return a[i].Value < a[j].Value }
