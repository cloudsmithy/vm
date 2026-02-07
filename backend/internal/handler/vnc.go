package handler

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// VNCWebSocket proxies a WebSocket connection to the VM's VNC port
func (h *Handler) VNCWebSocket(c *gin.Context) {
	name := c.Param("name")
	port, err := h.svc.GetVNCPort(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Connect to VNC server
	vncAddr := fmt.Sprintf("127.0.0.1:%d", port)
	vncConn, err := net.Dial("tcp", vncAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot connect to vnc: " + err.Error()})
		return
	}

	// Upgrade HTTP to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		vncConn.Close()
		return
	}

	// Bidirectional proxy with coordinated shutdown
	var once sync.Once
	closeAll := func() {
		once.Do(func() {
			vncConn.Close()
			ws.Close()
		})
	}
	var wg sync.WaitGroup
	var wsMu sync.Mutex

	// WS -> VNC
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer closeAll()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if _, err := vncConn.Write(msg); err != nil {
				return
			}
		}
	}()

	// VNC -> WS
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer closeAll()
		buf := make([]byte, 32*1024)
		for {
			n, err := vncConn.Read(buf)
			if err != nil {
				return
			}
			wsMu.Lock()
			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			wsMu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	wg.Wait()
}

// GetVNCPort returns the VNC port info for a VM
func (h *Handler) GetVNCPort(c *gin.Context) {
	port, err := h.svc.GetVNCPort(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"port": port})
}
