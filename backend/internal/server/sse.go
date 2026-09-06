package server

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) sseEvents(c *gin.Context) {
	id := c.Param("id")

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ch := s.bus.Subscribe(id)
	defer s.bus.Unsubscribe(id, ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case ev, ok := <-ch:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "data: %s\n\n", ev.JSON())
			c.Writer.Flush()
			return true
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
