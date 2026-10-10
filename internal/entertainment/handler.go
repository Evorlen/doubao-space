package entertainment

import (
	"errors"
	"net/http"
	"strconv"
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
	router.GET("/entertainment", h.list)
	router.POST("/entertainment", h.create)
	router.GET("/entertainment/:id", h.get)
	router.PUT("/entertainment/:id", h.update)
	router.DELETE("/entertainment/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	rating := 0
	if value := strings.TrimSpace(c.Query("rating")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writeError(c, &ValidationError{Message: "评分筛选格式不正确"})
			return
		}
		rating = parsed
	}
	response, err := h.service.List(c.Request.Context(), ListOptions{
		MediaType: c.Query("type"),
		Status:    c.Query("status"),
		Search:    c.Query("search"),
		Rating:    rating,
		Favorite:  parseBool(c.Query("favorite")),
		Sort:      c.Query("sort"),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) create(c *gin.Context) {
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &ValidationError{Message: "请求格式不正确"})
		return
	}
	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "作品 ID 不正确"})
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "作品 ID 不正确"})
		return
	}
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &ValidationError{Message: "请求格式不正确"})
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "作品 ID 不正确"})
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器暂时无法完成这个操作"
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = "没有找到这项作品"
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		message = validation.Message
	}
	c.JSON(status, gin.H{"error": message})
}
