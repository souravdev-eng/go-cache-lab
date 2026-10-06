package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// PUT /api/labs/consistency/books/:id
func (a *api) consistencyPut(c *gin.Context) {
	id, ok := bookID(c)
	if !ok {
		return
	}
	var payload struct {
		Title      string `json:"title"`
		Author     string `json:"author"`
		PriceCents *int64 `json:"price_cents"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Author) == "" || payload.PriceCents == nil || *payload.PriceCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, author, and price_cents are required; price_cents must be nonnegative"})
		return
	}
	input := bookstore.BookUpdate{Title: payload.Title, Author: payload.Author, PriceCents: *payload.PriceCents}
	book, err := a.store.Update(c.Request.Context(), id, input)
	if err != nil {
		storeError(c, err)
		return
	}
	// TODO(consistency): invalidate every lab cache after the database commit.
	c.JSON(http.StatusOK, book)
}
