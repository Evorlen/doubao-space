package schedules

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service      *Service
	resourcePath string
	resourceName string
}

func NewHandler(service *Service) *Handler {
	return newResourceHandler(service, "schedules", "日程")
}

func NewPlanHandler(service *Service) *Handler {
	return newResourceHandler(service, "plans", "计划")
}

func newResourceHandler(service *Service, resourcePath, resourceName string) *Handler {
	return &Handler{
		service:      service,
		resourcePath: resourcePath,
		resourceName: resourceName,
	}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	resourcePath := "/" + h.resourcePath
	itemPath := resourcePath + "/:id"
	router.GET(resourcePath, h.list)
	router.POST(resourcePath, h.create)
	router.GET(itemPath, h.get)
	router.PUT(itemPath, h.update)
	router.POST(itemPath+"/complete", h.complete)
	router.DELETE(itemPath, h.delete)
}

type schedulePayload struct {
	Title        string `json:"title"`
	Note         string `json:"note"`
	StartDate    string `json:"start_date"`
	IntervalDays int    `json:"interval_days"`
}

func (h *Handler) list(c *gin.Context) {
	plan, err := h.service.GetPlan(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) create(c *gin.Context) {
	payload, ok := bindPayload(c)
	if !ok {
		return
	}
	schedule, err := h.service.Create(c.Request.Context(), payload)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, schedule)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	schedule, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	payload, valid := bindPayload(c)
	if !valid {
		return
	}
	schedule, err := h.service.Update(c.Request.Context(), id, payload)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *Handler) complete(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	schedule, err := h.service.Complete(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindPayload(c *gin.Context) (ScheduleInput, bool) {
	var payload schedulePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式不正确"})
		return ScheduleInput{}, false
	}
	return ScheduleInput{
		Title:        payload.Title,
		Note:         payload.Note,
		StartDate:    payload.StartDate,
		IntervalDays: payload.IntervalDays,
	}, true
}

func (h *Handler) parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s ID 不正确", h.resourceName)})
		return 0, false
	}
	return id, true
}

func (h *Handler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器暂时无法完成这个操作"
	var validation *ValidationError
	var conflict *ConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = fmt.Sprintf("没有找到这项%s", h.resourceName)
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		message = validation.Message
	case errors.As(err, &conflict):
		status = http.StatusConflict
		message = conflict.Message
	}
	c.JSON(status, gin.H{"error": message})
}
