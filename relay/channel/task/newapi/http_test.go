package newapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchTaskHitsGenerationsPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/video/generations/cgt-1", r.URL.Path)
		assert.Equal(t, "Bearer k", r.Header.Get("Authorization"))
		w.Write([]byte(`{"status":"queued"}`))
	}))
	defer srv.Close()
	a := &TaskAdaptor{}
	resp, err := a.FetchTask(srv.URL, "k", map[string]any{"task_id": "cgt-1"}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
}
