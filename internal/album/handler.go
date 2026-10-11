package album

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const maxUploadRequestSize = MaxMediaSize + 1*1024*1024

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/doubao/album", h.list)
	router.POST("/doubao/album", h.upload)
	router.GET("/doubao/album/:id", h.get)
	router.PUT("/doubao/album/:id", h.update)
	router.GET("/doubao/album/:id/media", h.media)
	router.DELETE("/doubao/album/:id", h.delete)
}

type updatePayload struct {
	Note string `json:"note"`
}

func (h *Handler) list(c *gin.Context) {
	album, err := h.service.List(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, album)
}

func (h *Handler) upload(c *gin.Context) {
	if c.Request.ContentLength > maxUploadRequestSize {
		h.writeError(c, &ValidationError{Message: "单个照片或视频不能超过 100 MB。"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadRequestSize)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		message := "请选择要上传的照片或视频。"
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			message = "单个照片或视频不能超过 100 MB。"
		}
		h.writeError(c, &ValidationError{Message: message})
		return
	}
	defer file.Close()

	item, err := h.service.Create(c.Request.Context(), UploadInput{
		Name:     header.Filename,
		MimeType: header.Header.Get("Content-Type"),
		Note:     c.PostForm("note"),
		Reader:   file,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var payload updatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.writeError(c, &ValidationError{Message: "请求格式不正确。"})
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, UpdateInput{Note: payload.Note})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) media(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, path, err := h.service.GetMedia(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Header("Content-Type", item.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", urlEscape(item.Name)))
	c.File(path)
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "相册条目 ID 不正确。"})
		return 0, false
	}
	return id, true
}

func (h *Handler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器暂时无法完成这个操作。"
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		message = "没有找到这项相册记录。"
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		message = validation.Message
	}
	c.JSON(status, gin.H{"error": message})
}

func urlEscape(value string) string {
	const hexChars = "0123456789ABCDEF"
	result := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		char := value[i]
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			result = append(result, char)
			continue
		}
		result = append(result, '%', hexChars[char>>4], hexChars[char&0x0f])
	}
	return string(result)
}
