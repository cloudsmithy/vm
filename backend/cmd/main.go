package main

import (
	"log"
	"kvmmm/internal/handler"
	"kvmmm/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	svc, err := service.NewLibvirtService()
	if err != nil {
		log.Fatalf("连接 libvirt 失败: %v", err)
	}
	defer svc.Close()

	h := handler.NewHandler(svc)

	r := gin.Default()
	r.Use(cors.Default())
	r.MaxMultipartMemory = 8 << 30

	api := r.Group("/api")
	{
		api.GET("/host/info", h.GetHostInfo)
		api.GET("/host/nics", h.ListPhysicalNICs)

		// VM CRUD + actions
		api.GET("/vms", h.ListVMs)
		api.GET("/vms/:name", h.GetVM)
		api.GET("/vms/:name/detail", h.GetVMDetail)
		api.POST("/vms", h.CreateVM)
		api.PUT("/vms/:name", h.UpdateVM)
		api.DELETE("/vms/:name", h.DeleteVM)
		api.POST("/vms/:name/start", h.StartVM)
		api.POST("/vms/:name/shutdown", h.ShutdownVM)
		api.POST("/vms/:name/destroy", h.DestroyVM)
		api.POST("/vms/:name/reboot", h.RebootVM)
		api.POST("/vms/:name/suspend", h.SuspendVM)
		api.POST("/vms/:name/resume", h.ResumeVM)
		api.POST("/vms/:name/clone", h.CloneVM)
		api.GET("/vms/:name/autostart", h.GetAutostart)
		api.PUT("/vms/:name/autostart", h.SetAutostart)
		api.POST("/vms/:name/rename", h.RenameVM)
		api.POST("/vms/import", h.ImportVM)
		api.POST("/vms/batch", h.BatchAction)

		// VM devices
		api.POST("/vms/:name/disks", h.AttachDisk)
		api.DELETE("/vms/:name/disks/:target", h.DetachDisk)
		api.POST("/vms/:name/nics", h.AttachNIC)
		api.DELETE("/vms/:name/nics/:mac", h.DetachNIC)
		api.POST("/vms/:name/iso", h.AttachISO)
		api.DELETE("/vms/:name/iso", h.DetachISO)
		api.POST("/vms/:name/finish-install", h.FinishInstall)

		// VNC
		api.GET("/vms/:name/vnc", h.GetVNCPort)

		// Snapshots
		api.GET("/vms/:name/snapshots", h.ListSnapshots)
		api.POST("/vms/:name/snapshots", h.CreateSnapshot)
		api.DELETE("/vms/:name/snapshots/:snap", h.DeleteSnapshot)
		api.POST("/vms/:name/snapshots/:snap/revert", h.RevertSnapshot)
		api.POST("/vms/:name/snapshots/:snap/revert-to-new", h.RevertSnapshotToNew)

		// Networks
		api.GET("/networks", h.ListNetworks)
		api.POST("/networks", h.CreateNetwork)
		api.POST("/networks/:name/start", h.StartNetwork)
		api.POST("/networks/:name/stop", h.StopNetwork)
		api.DELETE("/networks/:name", h.DeleteNetwork)

		// Storage pools
		api.GET("/storage-pools", h.ListStoragePools)
		api.POST("/storage-pools", h.CreateStoragePool)
		api.POST("/storage-pools/:name/start", h.StartStoragePool)
		api.POST("/storage-pools/:name/stop", h.StopStoragePool)
		api.DELETE("/storage-pools/:name", h.DeleteStoragePool)

		// Storage volumes
		api.GET("/storage-pools/:name/volumes", h.ListVolumes)
		api.POST("/storage-volumes", h.CreateVolume)
		api.DELETE("/storage-pools/:name/volumes/:vol", h.DeleteVolume)

		// ISO
		api.GET("/isos", h.ListISOs)
		api.POST("/isos/upload", h.UploadISO)
		api.DELETE("/isos/:name", h.DeleteISO)
	}

	r.GET("/ws/vnc/:name", h.VNCWebSocket)

	log.Println("后端启动在 :8080")
	r.Run(":8080")
}
