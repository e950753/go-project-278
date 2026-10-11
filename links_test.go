package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/e950753/go-project-278/internal/db"
)

const testBaseURL = "http://short.test"

// setupTestRouter подключается к тестовой БД и очищает таблицу links.
func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_ = godotenv.Load()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, "TRUNCATE links RESTART IDENTITY")
	require.NoError(t, err)

	return setupRouter(&App{q: db.New(pool), baseURL: testBaseURL})
}

func doRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func createTestLink(t *testing.T, r http.Handler, body string) linkResponse {
	t.Helper()
	w := doRequest(r, http.MethodPost, "/api/links", body)
	require.Equal(t, http.StatusCreated, w.Code)
	var l linkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &l))
	return l
}

func linkPath(id int64) string {
	return "/api/links/" + strconv.FormatInt(id, 10)
}

func TestCreateLink(t *testing.T) {
	r := setupTestRouter(t)

	l := createTestLink(t, r, `{"original_url":"https://example.com","short_name":"exmpl"}`)

	assert.Equal(t, int64(1), l.ID)
	assert.Equal(t, "https://example.com", l.OriginalURL)
	assert.Equal(t, "exmpl", l.ShortName)
	assert.Equal(t, testBaseURL+"/r/exmpl", l.ShortURL)
}

func TestCreateLinkGeneratesShortName(t *testing.T) {
	r := setupTestRouter(t)

	l := createTestLink(t, r, `{"original_url":"https://example.com"}`)

	assert.Len(t, l.ShortName, 8)
	assert.Equal(t, testBaseURL+"/r/"+l.ShortName, l.ShortURL)
}

func TestCreateLinkDuplicateShortName(t *testing.T) {
	r := setupTestRouter(t)
	body := `{"original_url":"https://example.com","short_name":"dup"}`
	createTestLink(t, r, body)

	w := doRequest(r, http.MethodPost, "/api/links", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.JSONEq(t, `{"errors":{"short_name":"short name already in use"}}`, w.Body.String())
}

func TestCreateLinkInvalidJSON(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodPost, "/api/links", `{broken`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"invalid request"}`, w.Body.String())
}

func TestListLinks(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodGet, "/api/links", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `[]`, w.Body.String())

	createTestLink(t, r, `{"original_url":"https://a.com","short_name":"aaa"}`)
	createTestLink(t, r, `{"original_url":"https://b.com","short_name":"bbb"}`)

	w = doRequest(r, http.MethodGet, "/api/links", "")
	require.Equal(t, http.StatusOK, w.Code)
	var links []linkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &links))
	require.Len(t, links, 2)
	assert.Equal(t, "aaa", links[0].ShortName)
	assert.Equal(t, "bbb", links[1].ShortName)
}

func TestGetLink(t *testing.T) {
	r := setupTestRouter(t)
	created := createTestLink(t, r, `{"original_url":"https://example.com","short_name":"exmpl"}`)

	w := doRequest(r, http.MethodGet, linkPath(created.ID), "")

	require.Equal(t, http.StatusOK, w.Code)
	var got linkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, created, got)
}

func TestGetLinkNotFound(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodGet, "/api/links/999", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateLink(t *testing.T) {
	r := setupTestRouter(t)
	created := createTestLink(t, r, `{"original_url":"https://example.com","short_name":"exmpl"}`)

	w := doRequest(r, http.MethodPut, linkPath(created.ID),
		`{"original_url":"https://example.org","short_name":"new"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var got linkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "https://example.org", got.OriginalURL)
	assert.Equal(t, "new", got.ShortName)
	assert.Equal(t, testBaseURL+"/r/new", got.ShortURL)
}

func TestUpdateLinkNotFound(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodPut, "/api/links/999",
		`{"original_url":"https://example.org","short_name":"new"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateLinkDuplicateShortName(t *testing.T) {
	r := setupTestRouter(t)
	createTestLink(t, r, `{"original_url":"https://a.com","short_name":"aaa"}`)
	second := createTestLink(t, r, `{"original_url":"https://b.com","short_name":"bbb"}`)

	w := doRequest(r, http.MethodPut, linkPath(second.ID),
		`{"original_url":"https://b.com","short_name":"aaa"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteLink(t *testing.T) {
	r := setupTestRouter(t)
	created := createTestLink(t, r, `{"original_url":"https://example.com","short_name":"exmpl"}`)

	w := doRequest(r, http.MethodDelete, linkPath(created.ID), "")
	assert.Equal(t, http.StatusNoContent, w.Code)

	w = doRequest(r, http.MethodGet, linkPath(created.ID), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteLinkNotFound(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodDelete, "/api/links/999", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListLinksPagination(t *testing.T) {
	r := setupTestRouter(t)
	for i := 0; i < 7; i++ {
		createTestLink(t, r, `{"original_url":"https://example.com","short_name":"link`+strconv.Itoa(i)+`"}`)
	}

	w := doRequest(r, http.MethodGet, "/api/links?range=[0,5]", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 0-5/7", w.Header().Get("Content-Range"))
	var page []linkResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	assert.Len(t, page, 5)

	w = doRequest(r, http.MethodGet, "/api/links?range=[5,10]", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 5-10/7", w.Header().Get("Content-Range"))
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	assert.Len(t, page, 2)
}

func TestListLinksInvalidRange(t *testing.T) {
	r := setupTestRouter(t)

	w := doRequest(r, http.MethodGet, "/api/links?range=abc", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
