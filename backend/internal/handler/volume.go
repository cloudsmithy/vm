package handler

import (
	"net/http"

	"virtpanel/internal/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListVolumes(c *gin.Context) {
	vols, err := h.svc.ListVolumes(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vols)
}

func (h *Handler) CreateVolume(c *gin.Context) {
	var req model.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateVolume(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *Handler) DeleteVolume(c *gin.Context) {
	if err := h.svc.DeleteVolume(c.Param("name"), c.Param("vol")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
