package calendar

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/calendar", h.list)
}

func (h *Handler) list(c *gin.Context) {
	response, err := h.service.List(c.Request.Context(), ListOptions{
		Start:            c.Query("start"),
		End:              c.Query("end"),
		Sources:          parseSources(c.Query("sources")),
		IncludeCompleted: parseBool(c.Query("include_completed")),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func parseSources(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []string{value}
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (h *Handler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "日历暂时无法读取。"
	var validation *ValidationError
	if errors.As(err, &validation) {
		status = http.StatusBadRequest
		message = validation.Message
	}
	c.JSON(status, gin.H{"error": message})
}
