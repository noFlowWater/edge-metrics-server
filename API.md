# Edge Metrics Server API Specification

## Overview

Edge Metrics Server는 edge-metrics-exporter 클라이언트를 위한 중앙 설정 관리 서버입니다.

- **Base URL**: `http://localhost:8081`
- **Content-Type**: `application/json`

---

## Endpoints

### GET /config

모든 디바이스의 설정 목록을 조회합니다.

**Request**
```
GET /config
```

**Response (200 OK)**
```json
{
  "configs": [
    {
      "device_id": "edge-01",
      "device_type": "jetson_orin",
      "port": 9100,
      "reload_port": 9101,
      "enabled_metrics": ["jetson_power_vdd_gpu_soc_watts"],
      "jetson": {"use_tegrastats": true}
    },
    {
      "device_id": "edge-02",
      "device_type": "raspberry_pi",
      "port": 9100,
      "reload_port": 9101
    }
  ],
  "total": 2
}
```

**Example**
```bash
curl http://localhost:8081/config
```

---

### GET /config/{device_id}

디바이스별 설정을 조회합니다.

**Request**
```
GET /config/{device_id}
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname (예: `edge-01`, `orin-desktop`) |

**Response (200 OK)**
```json
{
  "device_type": "jetson_orin",
  "port": 9100,
  "reload_port": 9101,
  "enabled_metrics": [
    "jetson_power_vdd_gpu_soc_watts",
    "jetson_temp_cpu_celsius"
  ],
  "jetson": {
    "use_tegrastats": true
  }
}
```

**Response (404 Not Found)**
```json
{
  "error": "Device not found",
  "device_id": "unknown-device",
  "message": "No configuration available for this device"
}
```

**Example**
```bash
curl http://localhost:8081/config/edge-01
```

---

### PUT /config/{device_id}

디바이스 설정을 생성하거나 업데이트합니다 (Upsert).

**Request**
```
PUT /config/{device_id}
Content-Type: application/json
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Request Body**
```json
{
  "device_type": "jetson_orin",
  "port": 9100,
  "reload_port": 9101,
  "enabled_metrics": [
    "jetson_power_vdd_gpu_soc_watts"
  ],
  "jetson": {
    "use_tegrastats": true
  }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| device_type | string | **Yes** | - | 디바이스 타입 |
| port | integer | No | 9100 | Prometheus 메트릭 서버 포트 |
| reload_port | integer | No | 9101 | 설정 리로드 트리거 포트 |
| enabled_metrics | array | No | null | 수집할 메트릭 목록 (null=전체) |
| * | object | No | - | 디바이스별 추가 설정 (shelly, jetson 등) |

**Response (200 OK) - 새 디바이스 등록**
```json
{
  "status": "registered",
  "device_id": "orin-desktop",
  "reload_triggered": false
}
```

**Response (200 OK) - 기존 디바이스 업데이트**
```json
{
  "status": "updated",
  "device_id": "edge-01",
  "reload_triggered": true
}
```

> **Note**: `reload_triggered`가 `true`이면 exporter의 `/reload` 엔드포인트가 호출되어 설정이 즉시 적용됩니다.

**Response (400 Bad Request)**
```json
{
  "error": "Missing required field",
  "message": "device_type is required"
}
```

**Example - 새 디바이스 등록**
```bash
curl -X PUT http://localhost:8081/config/orin-desktop \
  -H "Content-Type: application/json" \
  -d '{
    "device_type": "jetson_orin",
    "jetson": {"use_tegrastats": true}
  }'
```

---

### POST /config/{device_id}

새 디바이스를 등록합니다. 이미 존재하는 경우 409 Conflict를 반환합니다.

**Request**
```
POST /config/{device_id}
Content-Type: application/json
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Request Body**
```json
{
  "device_type": "jetson_orin",
  "port": 9100,
  "reload_port": 9101
}
```

**Response (201 Created)**
```json
{
  "status": "created",
  "device_id": "new-device"
}
```

**Response (409 Conflict)**
```json
{
  "error": "Device already exists",
  "device_id": "edge-01",
  "message": "Use PUT to update existing device"
}
```

**Example**
```bash
curl -X POST http://localhost:8081/config/new-device \
  -H "Content-Type: application/json" \
  -d '{"device_type": "raspberry_pi"}'
```

---

### PATCH /config/{device_id}

디바이스 설정을 부분 업데이트합니다. 전달된 필드만 변경됩니다.

**Request**
```
PATCH /config/{device_id}
Content-Type: application/json
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Request Body**
```json
{
  "port": 9200
}
```

> 변경하고자 하는 필드만 포함하면 됩니다. `null`을 전달하면 필드를 기본값으로 리셋하거나 삭제합니다.

**Response (200 OK)**
```json
{
  "status": "patched",
  "device_id": "edge-01",
  "reload_triggered": true
}
```

**Response (404 Not Found)**
```json
{
  "error": "Device not found",
  "device_id": "unknown-device",
  "message": "Use POST or PUT to create new device"
}
```

**Example**
```bash
curl -X PATCH http://localhost:8081/config/edge-01 \
  -H "Content-Type: application/json" \
  -d '{"port": 9200}'
```

---

### DELETE /config/{device_id}

디바이스 설정을 삭제합니다.

**Request**
```
DELETE /config/{device_id}
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Response (200 OK)**
```json
{
  "status": "deleted",
  "device_id": "edge-01"
}
```

**Response (404 Not Found)**
```json
{
  "error": "Device not found",
  "device_id": "unknown-device"
}
```

**Example**
```bash
curl -X DELETE http://localhost:8081/config/edge-01
```

---

### GET /health

서버 상태를 확인합니다.

**Request**
```
GET /health
```

**Response (200 OK)**
```json
{
  "status": "healthy",
  "service": "config-server",
  "version": "1.0.0"
}
```

**Example**
```bash
curl http://localhost:8081/health
```

---

### GET /devices

등록된 모든 디바이스와 상태를 조회합니다.

**Request**
```
GET /devices
```

**Response (200 OK)**
```json
{
  "devices": [
    {
      "device_id": "edge-01",
      "device_type": "jetson_orin",
      "ip_address": "192.168.1.10",
      "port": 9100,
      "reload_port": 9101,
      "status": "healthy",
      "last_seen": "2024-01-15T10:30:00Z"
    },
    {
      "device_id": "edge-02",
      "device_type": "jetson_xavier",
      "ip_address": "192.168.1.11",
      "port": 9100,
      "reload_port": 9101,
      "status": "unreachable",
      "error": "connection refused"
    }
  ],
  "total": 2,
  "healthy": 1,
  "unhealthy": 1
}
```

| Field | Type | Description |
|-------|------|-------------|
| devices | array | 디바이스 상태 목록 |
| total | integer | 전체 디바이스 수 |
| healthy | integer | 정상 디바이스 수 |
| unhealthy | integer | 비정상 디바이스 수 |

**Device Status Fields**

| Field | Type | Description |
|-------|------|-------------|
| device_id | string | 디바이스 ID |
| device_type | string | 디바이스 타입 |
| ip_address | string | 디바이스 IP 주소 |
| port | integer | 메트릭 서버 포트 |
| reload_port | integer | 리로드 트리거 포트 |
| status | string | healthy, unhealthy, unreachable, unknown |
| last_seen | string | 마지막 응답 시간 (healthy인 경우) |
| error | string | 에러 메시지 (비정상인 경우) |

**Example**
```bash
curl http://localhost:8081/devices
```

---

### GET /devices/{device_id}/status

특정 디바이스의 상태를 조회합니다.

**Request**
```
GET /devices/{device_id}/status
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Response (200 OK)**
```json
{
  "device_id": "edge-01",
  "device_type": "jetson_orin",
  "ip_address": "192.168.1.10",
  "port": 9100,
  "reload_port": 9101,
  "status": "healthy",
  "last_seen": "2024-01-15T10:30:00Z"
}
```

**Response (404 Not Found)**
```json
{
  "error": "Device not found",
  "device_id": "unknown-device"
}
```

**Example**
```bash
curl http://localhost:8081/devices/edge-01/status
```

---

### POST /devices/{device_id}/reload

특정 디바이스에 수동으로 reload를 트리거합니다.

**Request**
```
POST /devices/{device_id}/reload
```

| Parameter | Type | Location | Description |
|-----------|------|----------|-------------|
| device_id | string | path | 디바이스 hostname |

**Response (200 OK)**
```json
{
  "status": "reloaded",
  "device_id": "edge-01"
}
```

**Response (404 Not Found)**
```json
{
  "error": "Device not found",
  "device_id": "unknown-device"
}
```

**Response (503 Service Unavailable)**
```json
{
  "status": "failed",
  "device_id": "edge-01",
  "error": "connection refused"
}
```

**Example**
```bash
curl -X POST http://localhost:8081/devices/edge-01/reload
```

---

### POST /devices/reload

모든 디바이스에 일괄 reload를 트리거합니다.

**Request**
```
POST /devices/reload
```

**Response (200 OK)**
```json
{
  "results": [
    {
      "device_id": "edge-01",
      "status": "reloaded"
    },
    {
      "device_id": "edge-02",
      "status": "failed",
      "error": "connection refused"
    }
  ],
  "total": 2,
  "success": 1,
  "failed": 1
}
```

**Example**
```bash
curl -X POST http://localhost:8081/devices/reload
```

---

### GET /metrics/summary

전체 시스템 요약 통계를 조회합니다.

**Request**
```
GET /metrics/summary
```

**Response (200 OK)**
```json
{
  "total": 5,
  "healthy": 3,
  "unhealthy": 2,
  "by_device_type": {
    "jetson_orin": 2,
    "raspberry_pi": 2,
    "shelly": 1
  }
}
```

**Example**
```bash
curl http://localhost:8081/metrics/summary
```

---

## Device Types

지원되는 디바이스 타입:

| device_type | Description | Extra Config |
|-------------|-------------|--------------|
| `jetson_orin` | NVIDIA Jetson Orin | `jetson` |
| `jetson_xavier` | NVIDIA Jetson Xavier | `jetson` |
| `jetson` | Generic NVIDIA Jetson | `jetson` |
| `raspberry_pi` | Raspberry Pi | - |
| `orange_pi` | Orange Pi | - |
| `lattepanda` | LattePanda | - |
| `shelly` | Shelly smart plug | `shelly` |

---

## Extra Config Examples

### Jetson
```json
{
  "device_type": "jetson_orin",
  "jetson": {
    "use_tegrastats": true
  }
}
```

### Shelly
```json
{
  "device_type": "shelly",
  "shelly": {
    "host": "192.168.1.100",
    "switch_id": 0
  }
}
```

### INA260
```json
{
  "device_type": "jetson_orin",
  "ina260": {
    "i2c_address": "0x40"
  }
}
```

---

## Error Responses

모든 에러 응답은 다음 형식을 따릅니다:

```json
{
  "error": "Error type",
  "device_id": "device-id",
  "message": "Detailed error message"
}
```

| Status Code | Description |
|-------------|-------------|
| 200 | 성공 |
| 201 | 생성됨 (POST) |
| 400 | 잘못된 요청 (필수 필드 누락, 잘못된 JSON) |
| 404 | 디바이스를 찾을 수 없음 |
| 409 | 충돌 (이미 존재하는 디바이스) |
| 500 | 서버 내부 오류 |

---

## Database Schema

```sql
CREATE TABLE devices (
    device_id TEXT PRIMARY KEY,
    device_type TEXT NOT NULL,
    port INTEGER DEFAULT 9100,
    reload_port INTEGER DEFAULT 9101,
    enabled_metrics TEXT,    -- JSON array
    extra_config TEXT,       -- JSON object
    ip_address TEXT,         -- Auto-detected from request
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8081 | 서버 포트 |
| DB_PATH | ./config.db | SQLite 데이터베이스 경로 |

---

## Running the Server

```bash
# 기본 실행
./edge-metrics-server

# 환경변수 지정
PORT=8080 DB_PATH=/data/config.db ./edge-metrics-server
```

---

## Client Integration

### Exporter Auto-Registration Flow

```
1. Exporter 시작
2. GET /config/{hostname} → 404 (미등록)
3. Local config.yaml 로드
4. PUT /config/{hostname} → 200 {"status": "registered"}
5. 다음 시작 시 GET → 200 (등록된 설정 사용)
```

### Example Client Code (Python)

```python
import requests

# 설정 조회
response = requests.get(f"{server}/config/{device_id}")
config = response.json()

# 설정 등록/업데이트
requests.put(f"{server}/config/{device_id}", json=config)
```
