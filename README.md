
# checker

Проверяет скорость загрузки сайта и пишет метрики в формате prometheus.

Метрики доступны по адресу `/metrics` в формате Prometheus:

 - `website_load_time_seconds` - время последней проверки
 - `website_requests_total` - общее количество запросов
 - `website_failed_requests_total` - количество неудачных запросов
 - `website_tcp_handshake_seconds`  - время TCP handshake
-  `website_ttfb_seconds`  - время до первого байта (TTFB) 
-  `website_content_length_bytes`  - размер страницы
-  `website_last_status_code`  - HTTP статус код
-  `program_uptime_seconds`  - время работы программы

 **Graceful shutdown**:  
-   корректная обработка сигналов SIGINT/SIGTERM      
-   таймаут на завершение работы 5 секунд
        
**Health checks**:
Эндпоинт  `/health`  проверяет:
-   время последней проверки сайта
-   процент неудачных запросов
            
**Логирование**:
-   подробные логи с префиксом [SPEEDMON]
-   логирование всех этапов проверки
        
**Конфигурация**:
-   возможность легко изменить параметры проверки
-   таймауты запросов
        
**Безопасность**:
-   блокировки (sync.RWMutex) для безопасного доступа к метрикам
-   ограничение времени выполнения запросов
        
**Веб-интерфейс**:
-   Простая HTML страница с информацией о мониторе по адресу `http://localhost:8080/`

### Параметры запуска
**Стандартный запуск без аргументов**  (проверяет cloud.ru):

    go run checker.go

**Запуск с кастомным URL**:

    go run checker.go -url=https://example.com

**Дополнительные параметры**:

    go run checker.go -url=https://example.com -interval=60 -port=9090 -timeout=15

**Доступные аргументы:**

-   `-url`  - URL сайта для проверки (по умолчанию  [https://cloud.ru](https://cloud.ru/))
-   `-interval`  - интервал проверки в секундах (по умолчанию 30)
-   `-port`  - порт HTTP сервера (по умолчанию 8080)   
-   `-timeout`  - таймаут запроса в секундах (по умолчанию 10)


Для запуска выполнить `go run checker.go` с нужными аргументами и открыть `http://localhost:8080/` в браузере  или `http://localhost:8080/metrics` в Prometheus.
