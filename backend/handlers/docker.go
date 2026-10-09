package handlers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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

type dockerStatsRaw struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Stats struct {
			Cache        uint64 `json:"cache"`
			InactiveFile uint64 `json:"inactive_file"`
		} `json:"stats"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}

type containerStatsResult struct {
	id     string
	cpu    float64
	ramStr string
	ram    uint64
}

func (h *DockerHandler) getDockerClient() (*http.Client, string, error) {
	sockPath := config.AppConfig.DockerSocket
	if sockPath == "" {
		sockPath = "/var/run/docker.sock"
	}

	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		return nil, sockPath, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, proto, addr string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "unix", sockPath)
			},
		},
		Timeout: 10 * time.Second,
	}
	return client, sockPath, nil
}

func calculateCPUPercent(stats *dockerStatsRaw) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		if onlineCPUs == 0 {
			onlineCPUs = 1
		}
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		pct := (cpuDelta / systemDelta) * onlineCPUs * 100.0
		if pct > 0 {
			return math.Round(pct*100) / 100
		}
	}
	return 0.0
}

func calculateRAM(stats *dockerStatsRaw) (uint64, string) {
	mem := stats.MemoryStats.Usage
	if stats.MemoryStats.Stats.InactiveFile > 0 && stats.MemoryStats.Stats.InactiveFile < mem {
		mem -= stats.MemoryStats.Stats.InactiveFile
	} else if stats.MemoryStats.Stats.Cache > 0 && stats.MemoryStats.Stats.Cache < mem {
		mem -= stats.MemoryStats.Stats.Cache
	}
	if mem == 0 {
		return 0, ""
	}
	var s string
	if mem < 1024*1024*1024 {
		s = fmt.Sprintf("%.2fMB", float64(mem)/(1024*1024))
	} else {
		s = fmt.Sprintf("%.2fGB", float64(mem)/(1024*1024*1024))
	}
	return mem, s
}

func stripDockerLogHeaders(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var buf strings.Builder
	idx := 0
	for idx < len(data) {
		if idx+8 <= len(data) && (data[idx] == 1 || data[idx] == 2) && data[idx+1] == 0 && data[idx+2] == 0 && data[idx+3] == 0 {
			frameLen := int(binary.BigEndian.Uint32(data[idx+4 : idx+8]))
			idx += 8
			if idx+frameLen <= len(data) {
				buf.Write(data[idx : idx+frameLen])
				idx += frameLen
			} else {
				buf.Write(data[idx:])
				break
			}
		} else {
			buf.Write(data[idx:])
			break
		}
	}
	return buf.String()
}

func (h *DockerHandler) GetDockerContainers(c *gin.Context) {
	client, sockPath, err := h.getDockerClient()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"available":  false,
			"message":    fmt.Sprintf("Docker socket未挂载或不可用 (%s)", sockPath),
			"containers": []models.DockerContainerInfo{},
		})
		return
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

	// Concurrently query stats for running containers with 2s timeout
	var wg sync.WaitGroup
	statsChan := make(chan containerStatsResult, len(rawContainers))

	for _, raw := range rawContainers {
		if raw.State == "running" {
			wg.Add(1)
			go func(cid string) {
				defer wg.Done()
				cCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(cCtx, "GET", "http://localhost/containers/"+cid+"/stats?stream=false", nil)
				if err != nil {
					return
				}
				sResp, err := client.Do(req)
				if err != nil {
					return
				}
				defer sResp.Body.Close()

				var stats dockerStatsRaw
				if err := json.NewDecoder(sResp.Body).Decode(&stats); err == nil {
					cpu := calculateCPUPercent(&stats)
					ramBytes, ramStr := calculateRAM(&stats)
					statsChan <- containerStatsResult{
						id:     cid,
						cpu:    cpu,
						ramStr: ramStr,
						ram:    ramBytes,
					}
				}
			}(raw.ID)
		}
	}

	wg.Wait()
	close(statsChan)

	statsMap := make(map[string]containerStatsResult)
	for s := range statsChan {
		statsMap[s.id] = s
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

		st := statsMap[raw.ID]

		containers = append(containers, models.DockerContainerInfo{
			ID:          shortID,
			Names:       cleanedNames,
			Image:       raw.Image,
			State:       raw.State,
			Status:      raw.Status,
			Created:     raw.Created,
			CPUPercent:  st.cpu,
			RAMUsageStr: st.ramStr,
			RAMBytes:    st.ram,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"available":  true,
		"containers": containers,
	})
}

func (h *DockerHandler) StartContainer(c *gin.Context) {
	id := c.Param("id")
	client, _, err := h.getDockerClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker socket未挂载或不可用", "code": 503})
		return
	}

	req, err := http.NewRequest("POST", "http://localhost/containers/"+id+"/start", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动容器失败: " + err.Error(), "code": 500})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified {
		c.JSON(http.StatusOK, gin.H{"message": "容器启动成功", "code": 200})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	c.JSON(http.StatusBadRequest, gin.H{"error": "启动容器失败: " + string(body), "code": resp.StatusCode})
}

func (h *DockerHandler) StopContainer(c *gin.Context) {
	id := c.Param("id")
	client, _, err := h.getDockerClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker socket未挂载或不可用", "code": 503})
		return
	}

	req, err := http.NewRequest("POST", "http://localhost/containers/"+id+"/stop?t=10", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "停止容器失败: " + err.Error(), "code": 500})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified {
		c.JSON(http.StatusOK, gin.H{"message": "容器停止成功", "code": 200})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	c.JSON(http.StatusBadRequest, gin.H{"error": "停止容器失败: " + string(body), "code": resp.StatusCode})
}

func (h *DockerHandler) RestartContainer(c *gin.Context) {
	id := c.Param("id")
	client, _, err := h.getDockerClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker socket未挂载或不可用", "code": 503})
		return
	}

	req, err := http.NewRequest("POST", "http://localhost/containers/"+id+"/restart?t=10", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重启容器失败: " + err.Error(), "code": 500})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, gin.H{"message": "容器重启成功", "code": 200})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	c.JSON(http.StatusBadRequest, gin.H{"error": "重启容器失败: " + string(body), "code": resp.StatusCode})
}

func (h *DockerHandler) GetContainerLogs(c *gin.Context) {
	id := c.Param("id")
	tail := c.DefaultQuery("tail", "150")
	client, _, err := h.getDockerClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker socket未挂载或不可用", "code": 503})
		return
	}

	req, err := http.NewRequest("GET", "http://localhost/containers/"+id+"/logs?stdout=1&stderr=1&tail="+tail, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败: " + err.Error(), "code": 500})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	cleanLogs := stripDockerLogHeaders(body)
	c.JSON(http.StatusOK, gin.H{"logs": cleanLogs, "code": 200})
}
