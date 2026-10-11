package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/e950753/go-project-278/internal/db"
)

type visitResponse struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Status    int32     `json:"status"`
}

func (a *App) redirect(c *gin.Context) {
	l, err := a.q.GetLinkByShortName(c, c.Param("code"))
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	_, err = a.q.CreateLinkVisit(c, db.CreateLinkVisitParams{
		LinkID:    l.ID,
		Ip:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Referer:   c.Request.Referer(),
		Status:    http.StatusFound,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Redirect(http.StatusFound, l.OriginalUrl)
}

func (a *App) listVisits(c *gin.Context) {
	total, err := a.q.CountLinkVisits(c)
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

	visits, err := a.q.ListLinkVisitsPaged(c, db.ListLinkVisitsPagedParams{
		Limit:  end - start,
		Offset: start,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp := make([]visitResponse, 0, len(visits))
	for _, v := range visits {
		resp = append(resp, visitResponse{
			ID: v.ID, LinkID: v.LinkID, CreatedAt: v.CreatedAt.Time,
			IP: v.Ip, UserAgent: v.UserAgent, Referer: v.Referer, Status: v.Status,
		})
	}
	c.Header("Content-Range", fmt.Sprintf("link_visits %d-%d/%d", start, end, total))
	c.JSON(http.StatusOK, resp)
}
