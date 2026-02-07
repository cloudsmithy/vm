package handler

import (
	"net/http"

	"virtpanel/internal/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListNetworks(c *gin.Context) {
	nets, err := h.svc.ListNetworks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nets)
}

func (h *Handler) CreateNetwork(c *gin.Context) {
	var req model.CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateNetwork(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *Handler) StartNetwork(c *gin.Context) {
	if err := h.svc.StartNetwork(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "started"})
}

func (h *Handler) StopNetwork(c *gin.Context) {
	if err := h.svc.StopNetwork(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}

func (h *Handler) DeleteNetwork(c *gin.Context) {
	if err := h.svc.DeleteNetwork(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
