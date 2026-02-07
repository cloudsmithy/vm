package handler

import (
	"net/http"

	"virtpanel/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListPortForwards(c *gin.Context) {
	rules, err := h.svc.ListPortForwards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *Handler) AddPortForward(c *gin.Context) {
	var req service.PortForward
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AddPortForward(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) DeletePortForward(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeletePortForward(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
