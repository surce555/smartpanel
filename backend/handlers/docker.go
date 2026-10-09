package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/models"
)

type DockerHandler struct{}

type dockerContainerRaw struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Created int64    `json:"Created"`
}

func (h *DockerHandler) GetDockerContainers(c *gin.Context) {
	sockPath := config.AppConfig.DockerSocket
	if sockPath == "" {
		sockPath = "/var/run/docker.sock"
	}

	// Check if docker socket exists
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"available":  false,
			"message":    "Docker socket未挂载或不存在 (/var/run/docker.sock)",
			"containers": []models.DockerContainerInfo{},
		})
		return
	}

	// Create custom HTTP client dialing unix domain socket
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, proto, addr string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "unix", sockPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost/containers/json?all=1")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"available":  false,
			"message":    "连接 Docker 守护进程失败: " + err.Error(),
			"containers": []models.DockerContainerInfo{},
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"available":  false,
			"message":    "读取 Docker 响应失败",
			"containers": []models.DockerContainerInfo{},
		})
		return
	}

	var rawContainers []dockerContainerRaw
	if err := json.Unmarshal(body, &rawContainers); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"available":  false,
			"message":    "解析 Docker 容器数据失败",
			"containers": []models.DockerContainerInfo{},
		})
		return
	}

	containers := make([]models.DockerContainerInfo, 0, len(rawContainers))
	for _, raw := range rawContainers {
		cleanedNames := make([]string, 0, len(raw.Names))
		for _, n := range raw.Names {
			if len(n) > 0 && n[0] == '/' {
				cleanedNames = append(cleanedNames, n[1:])
			} else {
				cleanedNames = append(cleanedNames, n)
			}
		}

		shortID := raw.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		containers = append(containers, models.DockerContainerInfo{
			ID:      shortID,
			Names:   cleanedNames,
			Image:   raw.Image,
			State:   raw.State,
			Status:  raw.Status,
			Created: raw.Created,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"available":  true,
		"containers": containers,
	})
}
