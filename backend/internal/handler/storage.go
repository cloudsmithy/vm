package handler

import (
	"net/http"

	"virtpanel/internal/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListStoragePools(c *gin.Context) {
	pools, err := h.svc.ListStoragePools()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pools)
}

func (h *Handler) CreateStoragePool(c *gin.Context) {
	var req model.CreateStoragePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateStoragePool(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *Handler) StartStoragePool(c *gin.Context) {
	if err := h.svc.StartStoragePool(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "started"})
}

func (h *Handler) StopStoragePool(c *gin.Context) {
	if err := h.svc.StopStoragePool(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}

func (h *Handler) DeleteStoragePool(c *gin.Context) {
	if err := h.svc.DeleteStoragePool(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
