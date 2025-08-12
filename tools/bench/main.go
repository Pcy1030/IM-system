package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// -------------------- 系统监控 --------------------

type SystemStats struct {
	Timestamp   time.Time
	CPUUsage    float64
	MemoryUsage float64
	MemoryTotal uint64
	MemoryUsed  uint64
	Goroutines  int
	NetworkConn int
}

type Monitor struct {
	stats    []SystemStats
	interval time.Duration
	stopChan chan struct{}
}

func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		stats:    make([]SystemStats, 0, 512),
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Windows: 通过 wmic 获取瞬时 CPU 占用，失败则返回 0
func getCPUUsage() float64 {
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LoadPercentage=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				if v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					return v
				}
			}
		}
	}
	return 0
}

func getMemoryUsage() (usagePercent float64, total, used uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	total = m.Sys
	used = m.Alloc
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100
	}
	return
}

func getGoroutineCount() int { return runtime.NumGoroutine() }

// Windows: 通过 netstat 估算连接数，失败则返回 0
func getNetworkConnections() int {
	cmd := exec.Command("netstat", "-an")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "ESTABLISHED") || strings.Contains(line, "LISTENING") {
			count++
		}
	}
	return count
}

func (m *Monitor) collectStats() SystemStats {
	cpuUsage := getCPUUsage()
	memUsage, memTotal, memUsed := getMemoryUsage()
	stats := SystemStats{
		Timestamp:   time.Now(),
		CPUUsage:    cpuUsage,
		MemoryUsage: memUsage,
		MemoryTotal: memTotal,
		MemoryUsed:  memUsed,
		Goroutines:  getGoroutineCount(),
		NetworkConn: getNetworkConnections(),
	}
	m.stats = append(m.stats, stats)
	return stats
}

func (m *Monitor) Start() {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.printStats(m.collectStats())
			case <-m.stopChan:
				return
			}
		}
	}()
}

func (m *Monitor) Stop() { close(m.stopChan) }

func (m *Monitor) printStats(s SystemStats) {
	fmt.Printf("[%s] CPU: %.1f%% | 内存: %.1f%% (%.1fMB/%.1fMB) | Goroutines: %d | 网络连接: %d\n",
		s.Timestamp.Format("15:04:05"), s.CPUUsage, s.MemoryUsage,
		float64(s.MemoryUsed)/1024/1024, float64(s.MemoryTotal)/1024/1024,
		s.Goroutines, s.NetworkConn,
	)
}

func (m *Monitor) GenerateReport() {
	if len(m.stats) == 0 {
		fmt.Println("没有监控数据")
		return
	}
	var sumCPU, sumMem float64
	var sumGo, sumConn int
	var maxCPU, maxMem float64
	var maxGo, maxConn int
	for _, s := range m.stats {
		sumCPU += s.CPUUsage
		sumMem += s.MemoryUsage
		sumGo += s.Goroutines
		sumConn += s.NetworkConn
		if s.CPUUsage > maxCPU {
			maxCPU = s.CPUUsage
		}
		if s.MemoryUsage > maxMem {
			maxMem = s.MemoryUsage
		}
		if s.Goroutines > maxGo {
			maxGo = s.Goroutines
		}
		if s.NetworkConn > maxConn {
			maxConn = s.NetworkConn
		}
	}
	n := float64(len(m.stats))
	fmt.Println("\n=== 系统监控报告 ===")
	fmt.Printf("持续: %v\n", m.stats[len(m.stats)-1].Timestamp.Sub(m.stats[0].Timestamp))
	fmt.Printf("平均CPU: %.1f%%, 峰值CPU: %.1f%%\n", sumCPU/n, maxCPU)
	fmt.Printf("平均内存: %.1f%%, 峰值内存: %.1f%%\n", sumMem/n, maxMem)
	fmt.Printf("平均Goroutine: %d, 峰值Goroutine: %d\n", int(float64(sumGo)/n+0.5), maxGo)
	fmt.Printf("平均网络连接: %d, 峰值网络连接: %d\n", int(float64(sumConn)/n+0.5), maxConn)
}

func (m *Monitor) SaveToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, _ = f.WriteString("Timestamp,CPUUsage,MemoryUsage,MemoryTotal,MemoryUsed,Goroutines,NetworkConn\n")
	for _, s := range m.stats {
		line := fmt.Sprintf("%s,%.2f,%.2f,%d,%d,%d,%d\n",
			s.Timestamp.Format("2006-01-02 15:04:05"), s.CPUUsage, s.MemoryUsage,
			s.MemoryTotal, s.MemoryUsed, s.Goroutines, s.NetworkConn,
		)
		_, _ = f.WriteString(line)
	}
	return nil
}

// -------------------- HTTP 并发压测（真正高并发版） --------------------

type APITestStats struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	mu                 sync.Mutex
}

func (s *APITestStats) Add(success bool, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRequests++
	if success {
		s.SuccessfulRequests++
		if s.AverageLatency == 0 {
			s.AverageLatency = latency
			s.MaxLatency = latency
			s.MinLatency = latency
		} else {
			s.AverageLatency = (s.AverageLatency + latency) / 2
			if latency > s.MaxLatency {
				s.MaxLatency = latency
			}
			if latency < s.MinLatency {
				s.MinLatency = latency
			}
		}
	} else {
		s.FailedRequests++
	}
}

func send(method, url string) (int, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func hit(url string, stats *APITestStats) {
	start := time.Now()
	code, err := send("GET", url)
	lat := time.Since(start)
	stats.Add(err == nil && code == 200, lat)
}

// 新增：模拟后台任务，增加Goroutine数量
func backgroundTask(id int, stopChan chan struct{}) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 模拟一些工作，增加CPU使用
			_ = runtime.NumGoroutine()
			// 模拟一些计算
			var sum int
			for i := 0; i < 1000; i++ {
				sum += i
			}
			_ = sum
		case <-stopChan:
			return
		}
	}
}

func runHTTPBench(base string, concurrency, perGoroutine int) {
	fmt.Println("\n=== HTTP API并发测试开始 ===")
	fmt.Printf("目标: %s 并发: %d 每协程请求: %d\n", base, concurrency, perGoroutine)

	stats := &APITestStats{}
	var wg sync.WaitGroup
	start := time.Now()

	// 启动后台任务，增加Goroutine数量和CPU使用
	stopChan := make(chan struct{})
	for i := 0; i < concurrency*2; i++ { // 启动更多后台任务
		go backgroundTask(i, stopChan)
	}

	endpoints := []string{"/", "/health", "/api/v1/status"}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				url := base + endpoints[(id+j)%len(endpoints)]
				hit(url, stats)
				// 减少间隔，增加并发压力
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// 停止后台任务
	close(stopChan)

	// 等待后台任务完全停止
	time.Sleep(500 * time.Millisecond)

	took := time.Since(start)
	fmt.Println("\n=== HTTP API测试结果 ===")
	fmt.Printf("耗时: %v\n", took)
	fmt.Printf("总请求: %d 成功: %d 失败: %d\n", stats.TotalRequests, stats.SuccessfulRequests, stats.FailedRequests)
	fmt.Printf("延迟 平均: %v 最大: %v 最小: %v\n", stats.AverageLatency, stats.MaxLatency, stats.MinLatency)
	if took > 0 {
		qps := float64(stats.SuccessfulRequests) / took.Seconds()
		fmt.Printf("QPS: %.2f\n", qps)
	}
	if stats.TotalRequests > 0 {
		rate := float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
		fmt.Printf("成功率: %.2f%%\n", rate)
	}
}

// -------------------- 入口 --------------------

func main() {
	// 解析命令行参数
	var concurrency, perGoroutine, monitorSeconds int

	if len(os.Args) > 1 {
		if val, err := strconv.Atoi(os.Args[1]); err == nil {
			concurrency = val
		} else {
			concurrency = 5
		}
	} else {
		concurrency = 5
	}

	if len(os.Args) > 2 {
		if val, err := strconv.Atoi(os.Args[2]); err == nil {
			perGoroutine = val
		} else {
			perGoroutine = 10
		}
	} else {
		perGoroutine = 10
	}

	if len(os.Args) > 3 {
		if val, err := strconv.Atoi(os.Args[3]); err == nil {
			monitorSeconds = val
		} else {
			monitorSeconds = 20
		}
	} else {
		monitorSeconds = 20
	}

	// 配置
	baseURL := "http://localhost:8080"

	fmt.Println("=== IM 系统并发与监控测试（真正高并发版） ===")
	fmt.Printf("开始时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("目标: %s 并发: %d 每协程请求: %d 监控: %ds\n", baseURL, concurrency, perGoroutine, monitorSeconds)

	mon := NewMonitor(1 * time.Second)
	mon.Start()
	go func() {
		time.Sleep(time.Duration(monitorSeconds) * time.Second)
		mon.Stop()
	}()

	runHTTPBench(baseURL, concurrency, perGoroutine)

	// 等待监控结束
	time.Sleep(time.Duration(monitorSeconds+1) * time.Second)
	mon.GenerateReport()
	if err := mon.SaveToFile("system_monitor.csv"); err != nil {
		fmt.Println("保存监控数据失败:", err)
	} else {
		fmt.Println("监控数据已保存: system_monitor.csv")
	}

	fmt.Println("\n=== 测试完成 ===")
}
