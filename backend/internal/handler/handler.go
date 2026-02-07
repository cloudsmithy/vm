package handler

import (
	"net"
	"net/http"

	"virtpanel/internal/model"
	"virtpanel/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.LibvirtService
}

func NewHandler(svc *service.LibvirtService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListPhysicalNICs(c *gin.Context) {
	nics, err := net.Interfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var result []gin.H
	for _, n := range nics {
		if n.Flags&net.FlagLoopback != 0 || n.Flags&net.FlagBroadcast == 0 {
			continue
		}
		name := n.Name
		// Skip virtual/container interfaces
		skip := false
		for _, prefix := range []string{"vir", "vne", "vet", "br", "doc", "tap", "mac", "lzc", "lxc", "cni", "flannel", "cali", "tunl", "heiyu"} {
			if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
				skip = true
				break
			}
		}
		if skip || len(n.HardwareAddr) == 0 {
			continue
		}
		addrs, _ := n.Addrs()
		var ip string
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
		result = append(result, gin.H{"name": n.Name, "mac": n.HardwareAddr.String(), "ip": ip, "up": n.Flags&net.FlagUp != 0})
	}
	c.JSON(http.StatusOK, result)
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

func (h *Handler) GetAutostart(c *gin.Context) {
	v, err := h.svc.GetAutostart(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"autostart": v})
}

func (h *Handler) SetAutostart(c *gin.Context) {
	var req struct {
		Autostart bool `json:"autostart"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetAutostart(c.Param("name"), req.Autostart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) RenameVM(c *gin.Context) {
	var req model.RenameVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.RenameVM(c.Param("name"), req.NewName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "renamed"})
}

func (h *Handler) ImportVM(c *gin.Context) {
	var req model.ImportVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.ImportVM(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "imported"})
}

func (h *Handler) BatchAction(c *gin.Context) {
	var req model.BatchActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	valid := map[string]bool{"start": true, "shutdown": true, "destroy": true, "delete": true}
	if !valid[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}
	errors := map[string]string{}
	for _, name := range req.Names {
		var err error
		switch req.Action {
		case "start":
			err = h.svc.StartVM(name)
		case "shutdown":
			err = h.svc.ShutdownVM(name)
		case "destroy":
			err = h.svc.DestroyVM(name)
		case "delete":
			err = h.svc.DeleteVM(name)
		}
		if err != nil {
			errors[name] = err.Error()
		}
	}
	if len(errors) > 0 {
		c.JSON(http.StatusOK, gin.H{"message": "partial", "errors": errors})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
