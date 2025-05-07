# checker

Проверяет скорость загрузки сайта https://cloud.ru/ и пишет метрики в формате prometheus


Программа запускает HTTP-сервер на порту 8080 и каждые 30 секунд проверяется скорость загрузки сайта cloud.ru

Метрики доступны по адресу `/metrics` в формате Prometheus:

 - `website_load_time_seconds` - время последней проверки
 - `website_requests_total` - общее количество запросов
 - `website_failed_requests_total` - количество неудачных запросов

Для запуска выполнить `go run checker.go` и открыть `http://localhost:8080/metrics` в браузере или в Prometheus.
