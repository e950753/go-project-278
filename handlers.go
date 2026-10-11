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

	"encoding/json"
	"fmt"

	"github.com/e950753/go-project-278/internal/db"
	"github.com/go-playground/validator/v10"
)

type App struct {
	q       *db.Queries
	baseURL string
}

type linkPayload struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name" binding:"omitempty,min=3,max=32"`
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
	total, err := a.q.CountLinks(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	start, end, paged, err := parseRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if !paged {
		start, end = 0, int32(total)
		if end == 0 {
			end = 1
		}
	}

	links, err := a.q.ListLinksPaged(c, db.ListLinksPagedParams{
		Limit:  end - start,
		Offset: start,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp := make([]linkResponse, 0, len(links))
	for _, l := range links {
		resp = append(resp, a.toResponse(l))
	}
	c.Header("Content-Range", fmt.Sprintf("links %d-%d/%d", start, end, total))
	c.JSON(http.StatusOK, resp)
}

func (a *App) createLink(c *gin.Context) {
	var p linkPayload
	if !bindLink(c, &p) {
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
	if !bindLink(c, &p) {
		return
	}
	if p.ShortName == "" {
		c.JSON(http.StatusUnprocessableEntity,
			gin.H{"errors": gin.H{"short_name": "short_name is required"}})
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

func parseRange(c *gin.Context) (start, end int32, ok bool, err error) {
	raw := c.Query("range")
	if raw == "" {
		return 0, 0, false, nil
	}
	var r []int32
	if err := json.Unmarshal([]byte(raw), &r); err != nil || len(r) != 2 {
		return 0, 0, false, fmt.Errorf("invalid range")
	}
	if r[0] < 0 || r[1] <= r[0] {
		return 0, 0, false, fmt.Errorf("invalid range")
	}
	return r[0], r[1], true, nil
}

func bindLink(c *gin.Context, p *linkPayload) bool {
	err := c.ShouldBindJSON(p)
	if err == nil {
		return true
	}

	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := gin.H{}
		for _, fe := range verrs {
			name := fe.Field()
			switch name {
			case "OriginalURL":
				name = "original_url"
			case "ShortName":
				name = "short_name"
			}
			fields[name] = fe.Error()
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
		return false
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	return false
}
