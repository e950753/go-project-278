package main

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/e950753/go-project-278/internal/db"
)

type App struct {
	q       *db.Queries
	baseURL string
}

type linkPayload struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type linkResponse struct {
	ID          int64  `json:"id"`
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortURL    string `json:"short_url"`
}

func (a *App) toResponse(l db.Link) linkResponse {
	return linkResponse{
		ID:          l.ID,
		OriginalURL: l.OriginalUrl,
		ShortName:   l.ShortName,
		ShortURL:    a.baseURL + "/r/" + l.ShortName,
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomName(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[idx.Int64()]
	}
	return string(b)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return 0, false
	}
	return id, true
}

func (a *App) listLinks(c *gin.Context) {
	links, err := a.q.ListLinks(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	resp := make([]linkResponse, 0, len(links))
	for _, l := range links {
		resp = append(resp, a.toResponse(l))
	}
	c.JSON(http.StatusOK, resp)
}

func (a *App) createLink(c *gin.Context) {
	var p linkPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if p.ShortName == "" {
		p.ShortName = randomName(8)
	}
	l, err := a.q.CreateLink(c, db.CreateLinkParams{
		OriginalUrl: p.OriginalURL,
		ShortName:   p.ShortName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusUnprocessableEntity,
				gin.H{"errors": gin.H{"short_name": "short name already in use"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, a.toResponse(l))
}

func (a *App) getLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	l, err := a.q.GetLink(c, id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, a.toResponse(l))
}

func (a *App) updateLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var p linkPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	l, err := a.q.UpdateLink(c, db.UpdateLinkParams{
		ID:          id,
		OriginalUrl: p.OriginalURL,
		ShortName:   p.ShortName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusUnprocessableEntity,
				gin.H{"errors": gin.H{"short_name": "short name already in use"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, a.toResponse(l))
}

func (a *App) deleteLink(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	n, err := a.q.DeleteLink(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
