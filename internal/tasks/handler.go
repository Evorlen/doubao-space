package tasks

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
	router.GET("/tasks", h.list)
	router.POST("/tasks", h.create)
	router.GET("/tasks/:id", h.get)
	router.PUT("/tasks/:id", h.update)
	router.POST("/tasks/:id/complete", h.complete)
	router.POST("/tasks/:id/move", h.move)
	router.POST("/tasks/:id/attachments", h.addAttachment)
	router.GET("/tasks/:id/attachments/:attachmentId", h.downloadAttachment)
	router.DELETE("/tasks/:id/attachments/:attachmentId", h.deleteAttachment)
	router.DELETE("/tasks/:id", h.delete)
	router.GET("/tags", h.listTags)
	router.POST("/tags", h.createTag)
}

func (h *Handler) list(c *gin.Context) {
	includeCompleted := parseBool(c.Query("include_completed"))
	status := strings.TrimSpace(c.Query("status"))
	if status == "completed" {
		includeCompleted = true
	}
	tasks, err := h.service.List(c.Request.Context(), ListOptions{
		IncludeCompleted: includeCompleted,
		Search:           c.Query("search"),
		Status:           status,
		Priority:         c.Query("priority"),
		Tag:              c.Query("tag"),
		Sort:             c.Query("sort"),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) create(c *gin.Context) {
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &InputError{Message: "请求格式不正确。"})
		return
	}
	task, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	task, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &InputError{Message: "请求格式不正确。"})
		return
	}
	task, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) complete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	task, err := h.service.Complete(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

type moveInput struct {
	ParentID *int64 `json:"parent_id"`
}

func (h *Handler) move(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input moveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &InputError{Message: "请求格式不正确。"})
		return
	}
	task, err := h.service.Move(c.Request.Context(), id, input.ParentID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) addAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeError(c, &InputError{Message: "请选择要上传的文件。"})
		return
	}
	defer file.Close()
	attachment, err := h.service.AddAttachment(c.Request.Context(), id, header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, attachment)
}

func (h *Handler) downloadAttachment(c *gin.Context) {
	taskID, ok := parseID(c)
	if !ok {
		return
	}
	attachmentID, err := strconv.ParseInt(c.Param("attachmentId"), 10, 64)
	if err != nil || attachmentID <= 0 {
		writeError(c, &InputError{Message: "附件 ID 不合法。"})
		return
	}
	attachment, path, err := h.service.GetAttachment(c.Request.Context(), taskID, attachmentID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Content-Type", attachment.MimeType)
	c.FileAttachment(path, attachment.Name)
}

func (h *Handler) deleteAttachment(c *gin.Context) {
	taskID, ok := parseID(c)
	if !ok {
		return
	}
	attachmentID, err := strconv.ParseInt(c.Param("attachmentId"), 10, 64)
	if err != nil || attachmentID <= 0 {
		writeError(c, &InputError{Message: "附件 ID 不合法。"})
		return
	}
	if err := h.service.DeleteAttachment(c.Request.Context(), taskID, attachmentID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listTags(c *gin.Context) {
	tags, err := h.service.ListTags(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, tags)
}

type tagInput struct {
	Name string `json:"name"`
}

func (h *Handler) createTag(c *gin.Context) {
	var input tagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, &InputError{Message: "请求格式不正确。"})
		return
	}
	tag, err := h.service.CreateTag(c.Request.Context(), input.Name)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, tag)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, &InputError{Message: "任务 ID 不合法。"})
		return 0, false
	}
	return id, true
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
	message := "服务器内部错误。"
	var inputErr *InputError
	var conflictErr *ConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = "任务不存在。"
	case errors.As(err, &inputErr):
		status = http.StatusBadRequest
		message = inputErr.Message
	case errors.As(err, &conflictErr):
		status = http.StatusConflict
		message = conflictErr.Message
	}
	c.JSON(status, gin.H{"error": message})
}
