package memos

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
	router.GET("/memos", h.list)
	router.POST("/memos", h.create)
	router.GET("/memos/:id", h.get)
	router.PUT("/memos/:id", h.update)
	router.POST("/memos/:id/archive", h.archive)
	router.POST("/memos/:id/restore", h.restore)
	router.DELETE("/memos/:id", h.delete)
}

func (h *Handler) list(c *gin.Context) {
	archived, err := parseArchived(c.Query("archived"))
	if err != nil {
		writeError(c, err)
		return
	}
	response, err := h.service.List(c.Request.Context(), ListOptions{
		Archived: archived,
		Search:   c.Query("search"),
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
	memo, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, memo)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "备忘录 ID 不正确"})
		return
	}
	memo, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, memo)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "备忘录 ID 不正确"})
		return
	}
	var input Input
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &ValidationError{Message: "请求格式不正确"})
		return
	}
	memo, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, memo)
}

func (h *Handler) archive(c *gin.Context) {
	h.changeArchiveState(c, true)
}

func (h *Handler) restore(c *gin.Context) {
	h.changeArchiveState(c, false)
}

func (h *Handler) changeArchiveState(c *gin.Context, archived bool) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "备忘录 ID 不正确"})
		return
	}
	memo, err := h.service.SetArchived(c.Request.Context(), id, archived)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, memo)
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		writeError(c, &ValidationError{Message: "备忘录 ID 不正确"})
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

func parseArchived(value string) (*bool, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return nil, nil
	}
	var archived bool
	switch value {
	case "1", "true", "yes", "on":
		archived = true
	case "0", "false", "no", "off":
		archived = false
	default:
		return nil, &ValidationError{Message: "归档筛选格式不正确"}
	}
	return &archived, nil
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器暂时无法完成这个操作"
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = "没有找到这条备忘录"
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		message = validation.Message
	}
	c.JSON(status, gin.H{"error": message})
}
