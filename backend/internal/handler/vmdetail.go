package handler

import (
	"net/http"
	"net/url"

	"virtpanel/internal/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetVMDetail(c *gin.Context) {
	detail, err := h.svc.GetVMDetail(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *Handler) AttachDisk(c *gin.Context) {
	var req model.AttachDiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AttachDisk(c.Param("name"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "attached"})
}

func (h *Handler) DetachDisk(c *gin.Context) {
	if err := h.svc.DetachDisk(c.Param("name"), c.Param("target")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "detached"})
}

func (h *Handler) AttachNIC(c *gin.Context) {
	var req model.AttachNICRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AttachNIC(c.Param("name"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "attached"})
}

func (h *Handler) DetachNIC(c *gin.Context) {
	mac := c.Param("mac")
	// Gin doesn't decode %3A in path params, handle URL-encoded MAC
	if decoded, err := url.PathUnescape(mac); err == nil {
		mac = decoded
	}
	if err := h.svc.DetachNIC(c.Param("name"), mac); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "detached"})
}

func (h *Handler) AttachISO(c *gin.Context) {
	var req model.AttachISORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AttachISO(c.Param("name"), req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "attached"})
}

func (h *Handler) DetachISO(c *gin.Context) {
	if err := h.svc.DetachISO(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "detached"})
}

func (h *Handler) CloneVM(c *gin.Context) {
	var req model.CloneVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CloneVM(c.Param("name"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cloned"})
}

func (h *Handler) FinishInstall(c *gin.Context) {
	if err := h.svc.FinishInstall(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
