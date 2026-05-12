package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

type PetHandler struct {
	svc *service.PetService
}

func NewPetHandler(svc *service.PetService) *PetHandler {
	return &PetHandler{svc: svc}
}

// GetPet returns the current user's pet.
func (h *PetHandler) GetPet(c *gin.Context) {
	userID := GetCurrentUserID(c)
	pet, err := h.svc.GetOrCreate(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// UpdatePet updates the current user's pet name.
func (h *PetHandler) UpdatePet(c *gin.Context) {
	userID := GetCurrentUserID(c)

	var req struct {
		Name string `json:"name" binding:"required,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, 10001, err.Error())
		return
	}

	pet, err := h.svc.GetOrCreate(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}

	pet.Name = req.Name
	if err := h.svc.Update(c.Request.Context(), pet); err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// FeedPet feeds the current user's pet.
func (h *PetHandler) FeedPet(c *gin.Context) {
	userID := GetCurrentUserID(c)
	pet, err := h.svc.Feed(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// PlayWithPet plays with the current user's pet.
func (h *PetHandler) PlayWithPet(c *gin.Context) {
	userID := GetCurrentUserID(c)
	pet, err := h.svc.Play(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, pet)
}

// GetInteractions returns interaction history for the current user's pet.
func (h *PetHandler) GetInteractions(c *gin.Context) {
	userID := GetCurrentUserID(c)
	limit := 20
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	interactions, err := h.svc.GetInteractions(c.Request.Context(), userID, limit)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, interactions)
}
