package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	"backend/internal/router"
	"backend/internal/seed"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// NewRouter cleans the database and returns a fresh router wired to it,
// ready for a single test.
func NewRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	require.NoError(t, seed.CleanDatabase(db))
	return router.New(config.TestConfig(), db)
}

// DoJSON serializes body as JSON (if non-nil) and performs the request
// against r, returning the recorder.
func DoJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// DoJSONAuth is DoJSON with an Authorization: Bearer header attached.
func DoJSONAuth(t *testing.T, r http.Handler, method, path, accessToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// DecodeData unmarshals rec.Body into a map and returns the "data" field.
func DecodeData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, _ := body["data"].(map[string]any)
	return data
}

// DecodeError unmarshals rec.Body into a map and returns the "error" field.
func DecodeError(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errObj, _ := body["error"].(map[string]any)
	return errObj
}

// DecodeDataList unmarshals rec.Body and returns the "data" field as a list,
// for endpoints whose data payload is a JSON array rather than an object.
func DecodeDataList(t *testing.T, rec *httptest.ResponseRecorder) []any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	list, _ := body["data"].([]any)
	return list
}
