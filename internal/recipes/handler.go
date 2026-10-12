package recipes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/recipes", h.list)
	router.PUT("/recipes/order", h.reorder)
	router.POST("/recipes/foods", h.createFood)
	router.PUT("/recipes/foods/:id", h.updateFood)
	router.DELETE("/recipes/foods/:id", h.deleteFood)
}

type foodPayload struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Note     string `json:"note"`
	Days     []int  `json:"days"`
}

type reorderPayload struct {
	FoodIDs       []int64 `json:"food_ids"`
	WeekdayValues []int   `json:"weekday_values"`
}

func (h *Handler) list(c *gin.Context) {
	plan, err := h.service.GetPlan(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) reorder(c *gin.Context) {
	var payload reorderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式不正确"})
		return
	}
	plan, err := h.service.Reorder(c.Request.Context(), payload.FoodIDs, payload.WeekdayValues)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) createFood(c *gin.Context) {
	var payload foodPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式不正确"})
		return
	}
	food, err := h.service.CreateFood(c.Request.Context(), FoodInput{
		Name:     payload.Name,
		Category: payload.Category,
		Note:     payload.Note,
		Days:     payload.Days,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, food)
}

func (h *Handler) updateFood(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "食物 ID 不正确"})
		return
	}
	var payload foodPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式不正确"})
		return
	}
	food, err := h.service.UpdateFood(c.Request.Context(), id, FoodInput{
		Name:     payload.Name,
		Category: payload.Category,
		Note:     payload.Note,
		Days:     payload.Days,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, food)
}

func (h *Handler) deleteFood(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "食物 ID 不正确"})
		return
	}
	if err := h.service.DeleteFood(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器暂时无法完成这个操作"
	var validation *ValidationError
	var conflict *ConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = "没有找到这项食物"
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		message = validation.Message
	case errors.As(err, &conflict):
		status = http.StatusConflict
		message = conflict.Message
	}
	c.JSON(status, gin.H{"error": message})
}
