package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config содержит настройки приложения
type Config struct {
	TargetURL      string
	CheckInterval  time.Duration
	ServerPort     string
	RequestTimeout time.Duration
}

// Metrics содержит все собираемые метрики
type Metrics struct {
	sync.RWMutex
	TotalRequests      int64
	FailedRequests     int64
	LastLoadTime       float64
	LastTCPHandshake   float64
	LastTTFB           float64
	LastContentLength  int64
	LastStatusCode     int
	ProgramStartTime   time.Time
	LastCheckTime      time.Time
}

var (
	config  Config
	metrics = Metrics{
		ProgramStartTime: time.Now(),
	}
	logger = log.New(os.Stdout, "[SPEEDMON] ", log.LstdFlags|log.Lmsgprefix)
)

func main() {
	// Парсим аргументы командной строки
	targetURL := flag.String("url", "https://cloud.ru", "URL сайта для мониторинга")
	checkInterval := flag.Int("interval", 30, "Интервал проверки в секундах")
	serverPort := flag.String("port", "8080", "Порт для HTTP сервера")
	requestTimeout := flag.Int("timeout", 10, "Таймаут запроса в секундах")

	flag.Parse()

	// Инициализируем конфиг
	config = Config{
		TargetURL:      *targetURL,
		CheckInterval:  time.Duration(*checkInterval) * time.Second,
		ServerPort:     *serverPort,
		RequestTimeout: time.Duration(*requestTimeout) * time.Second,
	}

	logger.Printf("Starting monitoring for URL: %s", config.TargetURL)
	logger.Printf("Check interval: %v", config.CheckInterval)
	logger.Printf("Server port: %s", config.ServerPort)
	logger.Printf("Request timeout: %v", config.RequestTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем HTTP сервер
	server := &http.Server{Addr: ":" + config.ServerPort}
	go func() {
		http.HandleFunc("/metrics", metricsHandler)
		http.HandleFunc("/health", healthHandler)
		http.HandleFunc("/", homeHandler)

		logger.Printf("Starting server on :%s", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	// Запускаем мониторинг сайта
	go monitorWebsite(ctx)

	// Ожидаем сигнал завершения
	<-sigChan
	logger.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("Server shutdown error: %v", err)
	}

	cancel()
	logger.Println("Server stopped gracefully")
}

func monitorWebsite(ctx context.Context) {
	ticker := time.NewTicker(config.CheckInterval)
	defer ticker.Stop()

	// Первая проверка сразу при старте
	checkWebsiteSpeed()

	for {
		select {
		case <-ticker.C:
			checkWebsiteSpeed()
		case <-ctx.Done():
			logger.Println("Stopping website monitoring")
			return
		}
	}
}

func checkWebsiteSpeed() {
	metrics.Lock()
	metrics.TotalRequests++
	metrics.Unlock()

	start := time.Now()

	// Создаем кастомный транспорт для измерения времени TCP handshake
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			startDial := time.Now()
			conn, err := net.DialTimeout(network, addr, config.RequestTimeout)
			if err != nil {
				return nil, err
			}

			metrics.Lock()
			metrics.LastTCPHandshake = time.Since(startDial).Seconds()
			metrics.Unlock()

			return conn, nil
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.RequestTimeout,
	}

	req, err := http.NewRequest("GET", config.TargetURL, nil)
	if err != nil {
		logger.Printf("Error creating request: %v", err)
		recordFailure()
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Printf("Error fetching website: %v", err)
		recordFailure()
		return
	}
	defer resp.Body.Close()

	// Записываем время до первого байта (TTFB)
	metrics.Lock()
	metrics.LastTTFB = time.Since(start).Seconds()
	metrics.LastStatusCode = resp.StatusCode
	metrics.Unlock()

	// Читаем тело ответа для измерения полного времени загрузки и размера
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("Error reading response body: %v", err)
		recordFailure()
		return
	}

	metrics.Lock()
	metrics.LastLoadTime = time.Since(start).Seconds()
	metrics.LastContentLength = int64(len(body))
	metrics.LastCheckTime = time.Now()
	metrics.Unlock()

	logger.Printf(
		"Website check completed - Status: %d, TCP: %.3fs, TTFB: %.3fs, Total: %.3fs, Size: %d bytes",
		resp.StatusCode,
		metrics.LastTCPHandshake,
		metrics.LastTTFB,
		metrics.LastLoadTime,
		len(body),
	)
}

func recordFailure() {
	metrics.Lock()
	metrics.FailedRequests++
	metrics.LastCheckTime = time.Now()
	metrics.Unlock()
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics.RLock()
	defer metrics.RUnlock()

	uptime := time.Since(metrics.ProgramStartTime).Seconds()

	metricsOutput := fmt.Sprintf(`
# HELP website_requests_total Total number of requests
# TYPE website_requests_total counter
website_requests_total %d

# HELP website_failed_requests_total Total number of failed requests
# TYPE website_failed_requests_total counter
website_failed_requests_total %d

# HELP website_load_time_seconds Time taken to fully load the website
# TYPE website_load_time_seconds gauge
website_load_time_seconds %.3f

# HELP website_tcp_handshake_seconds Time taken for TCP handshake
# TYPE website_tcp_handshake_seconds gauge
website_tcp_handshake_seconds %.3f

# HELP website_ttfb_seconds Time to first byte (TTFB)
# TYPE website_ttfb_seconds gauge
website_ttfb_seconds %.3f

# HELP website_content_length_bytes Size of the webpage content in bytes
# TYPE website_content_length_bytes gauge
website_content_length_bytes %d

# HELP website_last_status_code Last HTTP status code received
# TYPE website_last_status_code gauge
website_last_status_code %d

# HELP website_last_check_time_seconds Timestamp of last check
# TYPE website_last_check_time_seconds gauge
website_last_check_time_seconds %d

# HELP program_uptime_seconds Program uptime in seconds
# TYPE program_uptime_seconds gauge
program_uptime_seconds %.0f
`,
		metrics.TotalRequests,
		metrics.FailedRequests,
		metrics.LastLoadTime,
		metrics.LastTCPHandshake,
		metrics.LastTTFB,
		metrics.LastContentLength,
		metrics.LastStatusCode,
		metrics.LastCheckTime.Unix(),
		uptime,
	)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(metricsOutput))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	metrics.RLock()
	defer metrics.RUnlock()

	// Проверяем, когда была последняя проверка
	if time.Since(metrics.LastCheckTime) > config.CheckInterval*2 {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("UNHEALTHY: No recent checks"))
		return
	}

	// Проверяем процент ошибок
	if metrics.TotalRequests > 10 && float64(metrics.FailedRequests)/float64(metrics.TotalRequests) > 0.5 {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("UNHEALTHY: High failure rate"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<html>
<head><title>Website Speed Monitor</title></head>
<body>
	<h1>Website Speed Monitor</h1>
	<p>Monitoring: %s</p>
	<p>Check interval: %v</p>
	<ul>
		<li><a href="/metrics">Metrics</a> (Prometheus format)</li>
		<li><a href="/health">Health Check</a></li>
	</ul>
	<footer>Uptime: %.1f minutes</footer>
</body>
</html>`,
		config.TargetURL,
		config.CheckInterval,
		time.Since(metrics.ProgramStartTime).Minutes(),
	)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
