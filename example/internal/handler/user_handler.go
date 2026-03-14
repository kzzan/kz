package handler

import (
	"net/http"

	"example/internal/models"
	"example/internal/service"
	"example/pkg/pagination"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type ListUserRequest struct {
	Page    int    `form:"page"    binding:"min=1"`
	Size    int    `form:"size"    binding:"min=1,max=100"`
	Keyword string `form:"keyword"`
	Sort    string `form:"sort"`
}

type GetUserRequest struct {
	ID string `uri:"id" binding:"required"`
}

type CreateUserRequest struct {
	// TODO: 填写创建字段
	// Name string `json:"name" binding:"required"`
}

type UpdateUserRequest struct {
	// TODO: 填写更新字段
	// Name string `json:"name"`
}

type UserResponse struct {
	// TODO: 填写响应字段
	// ID        uint      `json:"id"`
	// CreatedAt time.Time `json:"created_at"`
}

type ListUserResponse struct {
	Total int64              `json:"total"`
	List  []UserResponse `json:"list"`
}

func toUserResponse(m *models.User) UserResponse {
	return UserResponse{
		// TODO: 填写字段映射
		// ID:        m.ID,
		// CreatedAt: m.CreatedAt,
	}
}

type UserHandler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type userHandler struct {
	logger  *zerolog.Logger
	service service.UserService
}

func NewUserHandler(i do.Injector) (UserHandler, error) {
	return &userHandler{
		logger:  do.MustInvoke[*zerolog.Logger](i),
		service: do.MustInvoke[service.UserService](i),
	}, nil
}

func (h *userHandler) List(c *gin.Context) {
	var req ListUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.List(c.Request.Context(), pagination.Query{
		Page:    req.Page,
		Size:    req.Size,
		Keyword: req.Keyword,
		Sort:    req.Sort,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("List user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	list := make([]UserResponse, len(result.List))
	for i, m := range result.List {
		list[i] = toUserResponse(&m)
	}
	c.JSON(http.StatusOK, ListUserResponse{
		Total: result.Total,
		List:  list,
	})
}

func (h *userHandler) Get(c *gin.Context) {
	var req GetUserRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("id", req.ID).Msg("Get user failed")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toUserResponse(result)})
}

func (h *userHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m := &models.User{
		// TODO: 填写字段映射
		// Name: req.Name,
	}
	result, err := h.service.Create(c.Request.Context(), m)
	if err != nil {
		h.logger.Error().Err(err).Msg("Create user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": toUserResponse(result)})
}

func (h *userHandler) Update(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var uriReq GetUserRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m := &models.User{
		// TODO: 填写字段映射
		// Name: req.Name,
	}
	result, err := h.service.Update(c.Request.Context(), uriReq.ID, m)
	if err != nil {
		h.logger.Error().Err(err).Str("id", uriReq.ID).Msg("Update user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toUserResponse(result)})
}

func (h *userHandler) Delete(c *gin.Context) {
	var req GetUserRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Delete(c.Request.Context(), req.ID); err != nil {
		h.logger.Error().Err(err).Str("id", req.ID).Msg("Delete user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}