package handler

import (
	"net/http"

	"kvmmm/internal/model"
	"kvmmm/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.LibvirtService
}

func NewHandler(svc *service.LibvirtService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetHostInfo(c *gin.Context) {
	info, err := h.svc.GetHostInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) ListVMs(c *gin.Context) {
	vms, err := h.svc.ListVMs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vms)
}

func (h *Handler) GetVM(c *gin.Context) {
	vm, err := h.svc.GetVM(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vm)
}

func (h *Handler) CreateVM(c *gin.Context) {
	var req model.CreateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateVM(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *Handler) DeleteVM(c *gin.Context) {
	if err := h.svc.DeleteVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *Handler) StartVM(c *gin.Context) {
	if err := h.svc.StartVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "started"})
}

func (h *Handler) ShutdownVM(c *gin.Context) {
	if err := h.svc.ShutdownVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "shutdown"})
}

func (h *Handler) DestroyVM(c *gin.Context) {
	if err := h.svc.DestroyVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "destroyed"})
}

func (h *Handler) RebootVM(c *gin.Context) {
	if err := h.svc.RebootVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "rebooted"})
}

func (h *Handler) SuspendVM(c *gin.Context) {
	if err := h.svc.SuspendVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "suspended"})
}

func (h *Handler) ResumeVM(c *gin.Context) {
	if err := h.svc.ResumeVM(c.Param("name")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resumed"})
}

func (h *Handler) UpdateVM(c *gin.Context) {
	var req model.UpdateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateVM(c.Param("name"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
