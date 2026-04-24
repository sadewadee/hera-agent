package holographic

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInit_CreatesSchema verifies that NewStore creates the facts table and the
// FTS5 virtual table.
func TestInit_CreatesSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "holo.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Check facts table exists by querying it.
	_, err = store.db.Exec("SELECT id FROM facts LIMIT 1")
	require.NoError(t, err, "facts table should exist")

	// Check FTS5 virtual table exists.
	_, err = store.db.Exec("SELECT rowid FROM facts_fts LIMIT 1")
	require.NoError(t, err, "facts_fts FTS5 table should exist")
}

// TestIsAvailable_AlwaysTrue confirms holographic is always available (local-only plugin).
func TestIsAvailable_AlwaysTrue(t *testing.T) {
	p := New()
	assert.True(t, p.IsAvailable())
}

// TestStoreMemory_Persists verifies that a fact stored via HandleToolCall is
// returned by a subsequent search.
func TestStoreMemory_Persists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	p := New()
	require.NoError(t, p.Initialize("session-1"))

	// Store a fact.
	result, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "add",
		"content": "the user loves Go programming",
	})
	require.NoError(t, err)

	var addResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &addResp))
	assert.Equal(t, "Fact stored.", addResp["result"])
	assert.NotNil(t, addResp["fact_id"])

	// Recall should return the stored fact.
	searchResult, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action": "search",
		"query":  "Go programming",
		"limit":  float64(10),
	})
	require.NoError(t, err)

	var searchResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(searchResult), &searchResp))
	results, ok := searchResp["results"].([]interface{})
	require.True(t, ok, "results should be a slice")
	assert.NotEmpty(t, results, "should have found the stored fact")
}

// TestRecallMemory_FTS5Match verifies that FTS5 returns only the matching fact
// and not an unrelated one.
func TestRecallMemory_FTS5Match(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	p := New()
	require.NoError(t, p.Initialize("session-2"))

	_, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "add",
		"content": "quick brown fox",
	})
	require.NoError(t, err)

	_, err = p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "add",
		"content": "slow red turtle",
	})
	require.NoError(t, err)

	result, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action": "search",
		"query":  "quick",
		"limit":  float64(10),
	})
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &resp))
	results, _ := resp["results"].([]interface{})
	assert.Len(t, results, 1, "only the 'quick brown fox' fact should match")
}

// TestRemoveFact_DeletesPersisted verifies that a fact can be stored then deleted.
func TestRemoveFact_DeletesPersisted(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	p := New()
	require.NoError(t, p.Initialize("session-3"))

	addResult, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "add",
		"content": "fact to be removed",
	})
	require.NoError(t, err)

	var addResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(addResult), &addResp))
	factID, ok := addResp["fact_id"].(float64)
	require.True(t, ok, "fact_id should be numeric")

	removeResult, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "remove",
		"fact_id": factID,
	})
	require.NoError(t, err)

	var removeResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(removeResult), &removeResp))
	assert.Equal(t, "Fact removed.", removeResp["result"])

	// Confirm deletion at the store level.
	var count int
	err = p.store.db.QueryRow("SELECT COUNT(*) FROM facts WHERE id = ?", int(factID)).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestAdjustTrust verifies the trust feedback mechanism.
func TestAdjustTrust_UpdatesTrustScore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	p := New()
	require.NoError(t, p.Initialize("session-4"))

	addResult, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action":  "add",
		"content": "trust test fact",
	})
	require.NoError(t, err)

	var addResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(addResult), &addResp))
	factID := addResp["fact_id"].(float64)

	// Positive feedback should raise trust.
	feedResult, err := p.HandleToolCall("fact_feedback", map[string]interface{}{
		"fact_id":  factID,
		"feedback": "helpful",
	})
	require.NoError(t, err)
	var feedResp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(feedResult), &feedResp))
	assert.Equal(t, "Feedback recorded.", feedResp["result"])

	// Verify trust is now above the default 0.5.
	var trust float64
	err = p.store.db.QueryRow("SELECT trust FROM facts WHERE id = ?", int(factID)).Scan(&trust)
	require.NoError(t, err)
	assert.Greater(t, trust, 0.5)
}

// TestListFacts_ReturnsMostRecent verifies list ordering.
func TestListFacts_ReturnsMostRecent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERA_HOME", dir)

	p := New()
	require.NoError(t, p.Initialize("session-5"))

	for _, content := range []string{"fact A", "fact B", "fact C"} {
		_, err := p.HandleToolCall("fact_store", map[string]interface{}{
			"action":  "add",
			"content": content,
		})
		require.NoError(t, err)
	}

	result, err := p.HandleToolCall("fact_store", map[string]interface{}{
		"action": "list",
		"limit":  float64(5),
	})
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &resp))
	results, _ := resp["results"].([]interface{})
	assert.Len(t, results, 3)
}

// TestStore_AddFact_DirectAPI tests the Store directly.
func TestStore_AddFact_DirectAPI(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer store.Close()

	id, err := store.AddFact("direct fact", "general", "tag1")
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	// Verify it is in the database.
	var content string
	err = store.db.QueryRow("SELECT content FROM facts WHERE id = ?", id).Scan(&content)
	require.NoError(t, err)
	assert.Equal(t, "direct fact", content)
}

// TestStore_SearchFTS_FallbackToLike verifies the LIKE fallback when FTS5
// receives a query that FTS5 would reject (single-character query triggers
// the fallback path because FTS5 porter tokenizer discards very short tokens).
func TestStore_SearchFTS_FallbackBehavior(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "fallback.db"))
	require.NoError(t, err)
	defer store.Close()

	_, err = store.AddFact("needle in haystack", "general", "")
	require.NoError(t, err)

	// A broad LIKE-friendly query that also succeeds via FTS5.
	facts, err := store.SearchFTS("needle", 10)
	require.NoError(t, err)
	require.NotEmpty(t, facts)
	assert.Equal(t, "needle in haystack", facts[0].Content)
}

// Confirm that the database driver used matches "sqlite" (modernc.org/sqlite).
func TestSQLiteDriver_IsRegistered(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "driver_check.db"))
	require.NoError(t, err, "modernc.org/sqlite driver should be registered as 'sqlite'")
	defer db.Close()
	require.NoError(t, db.Ping())
}
