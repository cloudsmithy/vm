package handler

import (
	"net/http"

	"kvmmm/internal/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListSnapshots(c *gin.Context) {
	snaps, err := h.svc.ListSnapshots(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snaps)
}

func (h *Handler) CreateSnapshot(c *gin.Context) {
	var req model.CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateSnapshot(c.Param("name"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *Handler) DeleteSnapshot(c *gin.Context) {
	if err := h.svc.DeleteSnapshot(c.Param("name"), c.Param("snap")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) RevertSnapshot(c *gin.Context) {
	if err := h.svc.RevertSnapshot(c.Param("name"), c.Param("snap")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reverted"})
}
