package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
	"log"
)

var (
	totalRequests = 0
	failedRequests = 0
)

func main() {
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Website Speed Monitor</title></head>
			<body>
				<h1>Website Speed Monitor</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
		</html>`))
	})

	// Запускаем проверку сайта в фоне
	go monitorWebsite()

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func monitorWebsite() {
	ticker := time.NewTicker(30 * time.Second) // Проверяем каждые 30 секунд
	defer ticker.Stop()

	for range ticker.C {
		checkWebsiteSpeed()
	}
}

func checkWebsiteSpeed() {
	start := time.Now()
	totalRequests++

	resp, err := http.Get("https://cloud.ru")
	if err != nil {
		log.Printf("Error fetching website: %v", err)
		failedRequests++
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа для точного измерения
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		failedRequests++
		return
	}

	duration := time.Since(start).Seconds()
	log.Printf("Website loaded in %.2f seconds", duration)

	// Здесь можно добавить дополнительные метрики
	// Например, статус код, размер контента и т.д.
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем текущее время проверки
	now := time.Now().Unix()

	// Формируем метрики в формате Prometheus
	metrics := fmt.Sprintf(`
# HELP website_load_time_seconds Time taken to load the website
# TYPE website_load_time_seconds gauge
website_load_time_seconds %d
# HELP website_requests_total Total number of requests
# TYPE website_requests_total counter
website_requests_total %d
# HELP website_failed_requests_total Total number of failed requests
# TYPE website_failed_requests_total counter
website_failed_requests_total %d
`, now, totalRequests, failedRequests)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(metrics))
}
