package holographic

// Retriever wraps Store with higher-level retrieval strategies.
type Retriever struct {
	store *Store
}

// NewRetriever creates a retriever backed by the given store.
func NewRetriever(store *Store) *Retriever {
	return &Retriever{store: store}
}

// Search performs a full-text search for facts.
func (r *Retriever) Search(query string, limit int) ([]Fact, error) {
	return r.store.SearchFTS(query, limit)
}

// Probe retrieves all facts about a specific entity.
func (r *Retriever) Probe(entity string) ([]Fact, error) {
	return r.store.ProbeEntity(entity)
}
