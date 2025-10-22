# k6 Load Testing for Wallet API

## Установка k6

### Windows
```bash
choco install k6
```

### Linux
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### macOS
```bash
brew install k6
```

### Docker
```bash
docker pull grafana/k6
```

## Доступные тесты

### 1. Constant Load Test (постоянная нагрузка)
**Цель**: Проверить, что система держит 1000 RPS на один кошелек в течение 1 минуты.

```bash
k6 run constant_load.js

# С кастомным BASE_URL
k6 run -e BASE_URL=http://localhost:8080 constant_load.js
```

**Ожидаемые результаты**:
- 95% запросов < 500ms
- 99% запросов < 1000ms
- Менее 1% ошибок
- Отсутствие 5xx ошибок

---

### 2. Spike Test (пиковая нагрузка)
**Цель**: Проверить, как система ведет себя при резком росте нагрузки до 2000 RPS.

```bash
k6 run spike_test.js
```

**Этапы**:
1. Разогрев: 100 VUs (10s)
2. Базовая нагрузка: 100 VUs (20s)
3. **Spike**: 2000 VUs (10s)
4. Пик: 2000 VUs (30s)
5. Восстановление: 100 VUs (10s)
6. Остывание: 100 VUs (20s)

**Ожидаемые результаты**:
- 95% запросов < 1000ms
- Менее 5% ошибок
- Система восстанавливается после spike

---

### 3. Stress Test (стресс-тест)
**Цель**: Найти предел системы и точку отказа.

```bash
k6 run stress_test.js
```

**Этапы**:
1. 500 VUs (1m)
2. 1000 VUs (1m)
3. 2000 VUs (1m)
4. **3000 VUs** (1m)
5. Удержание: 3000 VUs (2m)
6. Остывание (1m)

**Ожидаемые результаты**:
- 99% запросов < 2000ms
- Менее 10% ошибок
- Определить максимальный RPS

---

### 4. Soak Test (длительный тест)
**Цель**: Проверить стабильность при длительной работе (поиск утечек памяти).

```bash
k6 run soak_test.js
```

**Длительность**: ~34 минуты
- Разогрев: 500 VUs (2m)
- **Основной тест**: 500 VUs (30m)
- Остывание (2m)

**Ожидаемые результаты**:
- Стабильное время ответа
- Отсутствие деградации производительности
- Отсутствие утечек памяти

---

### 5. Multi-Wallet Test (множество кошельков)
**Цель**: Проверить производительность при распределенной нагрузке на 10 кошельков.

```bash
k6 run multi_wallet_test.js
```

**Параметры**:
- 10 разных кошельков
- 100 VUs (2m)
- Случайное распределение запросов

**Ожидаемые результаты**:
- 95% запросов < 300ms
- Менее 1% ошибок
- Лучшая производительность, чем single wallet

---

## Запуск с Docker

```bash
# Constant load test
docker run --rm -i --network=host grafana/k6 run - <constant_load.js

# Spike test
docker run --rm -i --network=host grafana/k6 run - <spike_test.js

# С кастомными параметрами
docker run --rm -i --network=host \
  -e BASE_URL=http://localhost:8080 \
  grafana/k6 run - <constant_load.js
```

## Сохранение результатов

### JSON
```bash
k6 run --out json=results.json constant_load.js
```

### CSV
```bash
k6 run --out csv=results.csv constant_load.js
```

### InfluxDB + Grafana
```bash
# Запустить InfluxDB
docker run -d -p 8086:8086 influxdb:1.8

# Запустить тест с выводом в InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 constant_load.js

# Визуализация в Grafana
docker run -d -p 3000:3000 grafana/grafana
```

### k6 Cloud (SaaS)
```bash
# Регистрация: https://app.k6.io/
k6 login cloud

# Запуск с выводом в Cloud
k6 run --out cloud constant_load.js
```

## Интерпретация результатов

### Основные метрики

```
http_req_duration..............: avg=45ms    min=10ms  med=40ms  max=200ms  p(95)=95ms
```
- **avg**: среднее время ответа
- **med**: медиана (50 перцентиль)
- **p(95)**: 95% запросов быстрее этого значения
- **p(99)**: 99% запросов быстрее этого значения

```
http_req_failed................: 0.00%   ✓ 0         ✗ 50000
```
- **0.00%**: процент неудачных запросов
- **✓ 0**: количество успешных
- **✗ 50000**: количество неудачных

```
http_reqs......................: 50000   1666/s
```
- **50000**: всего запросов
- **1666/s**: запросов в секунду (RPS)

### Признаки проблем

❌ **Высокая задержка**:
```
http_req_duration: p(95)=5000ms
```
→ Проверить DB connections, advisory locks, retry logic

❌ **Много 5xx ошибок**:
```
http_req_failed: 15.00%
```
→ Проверить логи приложения, увеличить retry attempts

❌ **Деградация со временем** (soak test):
```
# Через 5 минут: p(95)=100ms
# Через 30 минут: p(95)=2000ms
```
→ Утечка памяти, проверить DB connection pool

## Рекомендации по оптимизации

### Если тесты падают:

1. **Увеличить HTTP timeout**:
```env
HTTP_TIMEOUT=60s
```

2. **Увеличить retry параметры**:
```env
RETRY_MAX_ATTEMPTS=15
RETRY_BASE_DELAY_MS=20
```

3. **Оптимизировать DB pool**:
```env
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=30
```

4. **Настроить PostgreSQL**:
```sql
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '512MB';
```

## CI/CD Integration

### GitHub Actions
```yaml
name: Load Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start services
        run: docker-compose up -d
      
      - name: Wait for services
        run: sleep 10
      
      - name: Run k6 load test
        run: |
          docker run --rm -i --network=host \
            grafana/k6 run - <tests/k6/constant_load.js
```

## Troubleshooting

**Проблема**: `connection refused`
```bash
# Проверить, что сервис запущен
curl http://localhost:8080/health

# Проверить Docker network
docker-compose ps
```

**Проблема**: `too many open files`
```bash
# Увеличить лимит (Linux)
ulimit -n 65535

# Постоянно (добавить в /etc/security/limits.conf)
* soft nofile 65535
* hard nofile 65535
```

**Проблема**: Медленные тесты
```bash
# Уменьшить VUs
k6 run --vus 100 --duration 30s constant_load.js

# Увеличить sleep между запросами
```

## Следующие шаги

После успешного прохождения тестов:

1. ✅ **Constant Load** → Базовая стабильность
2. ✅ **Multi-Wallet** → Масштабируемость
3. ✅ **Spike** → Устойчивость к пикам
4. ✅ **Stress** → Предел системы
5. ✅ **Soak** → Долгосрочная стабильность

Интегрируйте в CI/CD и мониторинг производительности!

