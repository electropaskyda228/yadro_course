package core

type Comics struct {
	ID  int
	URL string
}

type IndexInfoOne struct {
	Word       string
	Comics_ids []int
}

type IndexInfo struct {
	Comics []IndexInfoOne
}

type SearchReply struct {
	Comics []Comics
}

type SearchRequest struct {
	Limit  int
	Phrase string
}
