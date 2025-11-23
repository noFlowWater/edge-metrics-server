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

## Kubernetes Integration

### GET /kubernetes/status

전체 Kubernetes 동기화 상태를 조회합니다.

**Request**
```
GET /kubernetes/status?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| namespace | string | query | monitoring | 조회할 네임스페이스 |

**Response (200 OK)**
```json
{
  "kubernetes_enabled": true,
  "namespace": "monitoring",
  "total_k8s_resources": 5,
  "total_registered_devices": 7,
  "synced": 5,
  "unsynced": 2,
  "resources": [
    {
      "device_id": "edge-01",
      "service_exists": true,
      "endpoints_exists": true
    },
    {
      "device_id": "edge-02",
      "service_exists": false,
      "endpoints_exists": false
    }
  ]
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl http://localhost:8081/kubernetes/status?namespace=monitoring
```

---

### GET /kubernetes/health

Kubernetes 연결 상태 및 RBAC 권한을 확인합니다.

**Request**
```
GET /kubernetes/health?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| namespace | string | query | monitoring | 확인할 네임스페이스 |

**Response (200 OK)**
```json
{
  "kubernetes_available": true,
  "client_initialized": true,
  "namespace_accessible": true,
  "rbac_permissions": {
    "namespace": "ok",
    "services": "ok",
    "endpoints": "ok"
  }
}
```

**Response (503 Service Unavailable)**
```json
{
  "kubernetes_available": false,
  "client_initialized": false,
  "namespace_accessible": false,
  "rbac_permissions": {}
}
```

**Example**
```bash
curl http://localhost:8081/kubernetes/health
```

---

### POST /kubernetes/sync

현재 healthy 상태인 모든 디바이스를 Kubernetes Service + Endpoints로 동기화합니다.

**Request**
```
POST /kubernetes/sync
Content-Type: application/json
```

**Request Body**
```json
{
  "namespace": "monitoring"
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| namespace | string | No | monitoring | 동기화 대상 Kubernetes 네임스페이스 |

**Response (200 OK)**
```json
{
  "status": "synced",
  "created": [
    {
      "device_id": "edge-01",
      "service": "edge-device-edge-01",
      "status": "created"
    }
  ],
  "updated": [
    {
      "device_id": "edge-02",
      "service": "edge-device-edge-02",
      "status": "updated"
    }
  ],
  "deleted": [],
  "failed": [],
  "total_healthy": 2
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl -X POST http://localhost:8081/kubernetes/sync \
  -H "Content-Type: application/json" \
  -d '{"namespace": "monitoring"}'
```

**동작:**
1. GET /devices API를 호출하여 healthy 디바이스 목록 조회
2. 각 디바이스마다 Service + Endpoints 리소스 생성/업데이트
   - Service 이름: `edge-device-{device_id}`
   - Endpoints IP: 디바이스의 `ip_address`
   - 포트: 디바이스의 `port` (기본 9100)
   - 레이블: `app=edge-exporter`, `device_id`, `device_type`, `managed_by=edge-metrics-server`
3. DB에는 있지만 unhealthy하거나 삭제된 디바이스의 리소스는 삭제
4. 결과 반환

---

### POST /kubernetes/sync/{device_id}

특정 디바이스만 Kubernetes에 동기화합니다.

**Request**
```
POST /kubernetes/sync/{device_id}?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| device_id | string | path | - | 동기화할 디바이스 ID |
| namespace | string | query | monitoring | 동기화 대상 네임스페이스 |

**Response (200 OK)**
```json
{
  "device_id": "edge-01",
  "service": "edge-device-edge-01",
  "status": "created"
}
```

**Response (200 OK - Failed)**
```json
{
  "device_id": "edge-01",
  "service": "edge-device-edge-01",
  "status": "failed",
  "error": "device is not healthy"
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl -X POST http://localhost:8081/kubernetes/sync/edge-01?namespace=monitoring
```

---

### GET /kubernetes/manifests

Healthy 디바이스들의 Kubernetes YAML 매니페스트를 생성합니다 (수동 적용용).

**Request**
```
GET /kubernetes/manifests?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| namespace | string | query | monitoring | 매니페스트 생성 대상 네임스페이스 |

**Response (200 OK)**
```yaml
# Kubernetes manifests for edge devices
# Generated for namespace: monitoring

---
apiVersion: v1
kind: Service
metadata:
  name: edge-device-edge-01
  namespace: monitoring
  labels:
    app: edge-exporter
    device_id: edge-01
    device_type: jetson_orin
    managed_by: edge-metrics-server
spec:
  clusterIP: None
  ports:
  - name: metrics
    port: 9100
    targetPort: 9100
    protocol: TCP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: edge-device-edge-01
  namespace: monitoring
  labels:
    app: edge-exporter
    device_id: edge-01
    managed_by: edge-metrics-server
subsets:
- addresses:
  - ip: 192.168.1.10
  ports:
  - name: metrics
    port: 9100
    protocol: TCP

# ... (추가 디바이스들)
```

**Example**
```bash
# YAML 생성 및 저장
curl http://localhost:8081/kubernetes/manifests?namespace=monitoring > edge-devices.yaml

# Kubernetes에 적용
kubectl apply -f edge-devices.yaml
```

**동작:**
1. 모든 디바이스 설정 조회
2. 각 디바이스의 health 체크
3. Healthy 디바이스들만 YAML 매니페스트 생성
4. text/plain으로 반환

---

### GET /kubernetes/resources/{device_id}

특정 디바이스의 Kubernetes 리소스 상세 정보를 조회합니다.

**Request**
```
GET /kubernetes/resources/{device_id}?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| device_id | string | path | - | 조회할 디바이스 ID |
| namespace | string | query | monitoring | 조회할 네임스페이스 |

**Response (200 OK)**
```json
{
  "device_id": "edge-01",
  "service": {
    "name": "edge-device-edge-01",
    "exists": true,
    "cluster_ip": "None",
    "ports": [
      {
        "name": "metrics",
        "port": 9100
      }
    ]
  },
  "endpoints": {
    "name": "edge-device-edge-01",
    "exists": true,
    "ready_addresses": ["192.168.1.10:9100"],
    "not_ready_addresses": []
  },
  "prometheus_target": "http://edge-device-edge-01.monitoring.svc:9100/metrics"
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl http://localhost:8081/kubernetes/resources/edge-01?namespace=monitoring
```

---

### DELETE /kubernetes/resources/{device_id}

특정 디바이스의 Kubernetes 리소스를 삭제합니다.

**Request**
```
DELETE /kubernetes/resources/{device_id}?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| device_id | string | path | - | 삭제할 디바이스 ID |
| namespace | string | query | monitoring | 삭제할 네임스페이스 |

**Response (200 OK)**
```json
{
  "device_id": "edge-01",
  "service": "edge-device-edge-01",
  "status": "deleted"
}
```

**Response (200 OK - Failed)**
```json
{
  "device_id": "edge-01",
  "service": "edge-device-edge-01",
  "status": "failed",
  "error": "delete service: services \"edge-device-edge-01\" not found"
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl -X DELETE http://localhost:8081/kubernetes/resources/edge-01?namespace=monitoring
```

---

### DELETE /kubernetes/cleanup

monitoring 네임스페이스의 모든 edge-device-* 리소스를 삭제합니다.

**Request**
```
DELETE /kubernetes/cleanup?namespace=monitoring
```

| Parameter | Type | Location | Default | Description |
|-----------|------|----------|---------|-------------|
| namespace | string | query | monitoring | 정리할 네임스페이스 |

**Response (200 OK)**
```json
{
  "status": "cleaned",
  "deleted_services": [
    "edge-device-edge-01",
    "edge-device-edge-02"
  ],
  "deleted_endpoints": [
    "edge-device-edge-01",
    "edge-device-edge-02"
  ],
  "namespace": "monitoring"
}
```

**Response (503 Service Unavailable)**
```json
{
  "error": "Kubernetes client not initialized",
  "message": "Server not running in Kubernetes environment or kubeconfig not found"
}
```

**Example**
```bash
curl -X DELETE http://localhost:8081/kubernetes/cleanup?namespace=monitoring
```

**동작:**
1. `managed_by=edge-metrics-server` 레이블을 가진 모든 Service 조회
2. 모든 Service 삭제
3. `managed_by=edge-metrics-server` 레이블을 가진 모든 Endpoints 조회
4. 모든 Endpoints 삭제
5. 삭제된 리소스 목록 반환

---

## Device Types

지원되는 디바이스 타입:

| device_type | Description | Extra Config |
|-------------|-------------|--------------|
| `jetson_orin` | NVIDIA Jetson Orin | `jetson` |
| `jetson_xavier` | NVIDIA Jetson Xavier | `jetson` |
| `jetson_nano` | NVIDIA Jetson Nano | `jetson` |
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

## Grafana Visualization

### Overview

Grafana 대시보드를 통해 엣지 디바이스의 전력 메트릭을 실시간으로 시각화하고, 특정 시간 범위의 에너지 사용량 통계를 분석할 수 있습니다.

### Accessing Grafana

Grafana는 kube-prometheus-stack을 통해 monitoring 네임스페이스에 설치되어 있습니다.

**NodePort 접속 (외부에서 접속)**
```bash
# NodePort 확인
kubectl get svc -n monitoring monitoring-grafana

# 브라우저에서 접속
http://<NodeIP>:31932
```

**Port Forward 접속 (로컬에서 접속)**
```bash
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80

# 브라우저에서 접속
http://localhost:3000
```

**기본 로그인 정보**
- Username: `admin`
- Password: `prom-operator` (kube-prometheus-stack 기본값)

### Dashboard Import

**방법 1: JSON 파일로 Import**

1. Grafana에 로그인
2. 왼쪽 메뉴에서 `Dashboards` → `Import` 클릭
3. `Upload JSON file` 선택
4. `manifests/grafana-dashboard.json` 파일 업로드
5. Prometheus 데이터소스 선택 (기본: `prometheus`)
6. `Import` 버튼 클릭

**방법 2: JSON 내용 직접 붙여넣기**

1. `manifests/grafana-dashboard.json` 파일 내용 복사
2. Grafana에서 `Dashboards` → `Import` → `Import via panel json`
3. JSON 내용 붙여넣기
4. `Load` → `Import` 클릭

### Dashboard Panels

**1. 실시간 전력 메트릭 (Real-time Power Metrics)**
- 모든 디바이스의 전력 메트릭을 시계열 그래프로 표시
- 디바이스 타입별 필터링 가능 (`$device_type` 변수)
- 호스트명 필터링 가능 (`$hostname` 변수)
- Legend에 Last, Mean, Max 값 표시

**2. 총 에너지 사용량 (Total Energy Usage)**
- 선택한 시간 범위 내의 총 에너지 사용량 (Wh)
- 디바이스 타입별 (Xavier, Nano, Orin) 집계
- Stat 패널로 표시

**3. 평균/최대/최소 전력 (Average/Max/Min Power)**
- 선택한 시간 범위의 통계값
- 워크로드 분석 시 유용

**4. 디바이스별 에너지 사용 비율 (Energy Usage by Device)**
- Pie Chart로 디바이스별 에너지 사용 비중 표시
- 호스트명별 비교

**5. 디바이스별 전력/에너지 통계 (Power/Energy Statistics)**
- Table 형식으로 모든 통계를 한눈에 확인
- 디바이스별 평균/최대/최소 전력 및 총 에너지

### Dashboard Variables

대시보드 상단의 변수 선택기를 통해 필터링 가능:

- `$device_type`: 디바이스 타입 선택 (jetson_xavier, jetson_nano, jetson_orin)
- `$hostname`: 호스트명 선택 (V2X-GATEWAY, nano, orin-desktop 등)
- 기본값: `All` (모든 디바이스 표시)

### PromQL Query Examples

**실시간 전력 (Xavier)**
```promql
jetson_power_vdd_in_watts{device_type="jetson_xavier"}
```

**시간 범위 내 총 에너지 (Nano)**
```promql
sum(increase(jetson_power_pom_5v_in_watts{device_type="jetson_nano"}[$__range]) * $__range_s / 3600)
```

**평균 전력 (Orin)**
```promql
avg_over_time(jetson_power_vdd_gpu_soc_watts{device_type="jetson_orin"}[$__range]) +
avg_over_time(jetson_power_vdd_cpu_cv_watts{device_type="jetson_orin"}[$__range]) +
avg_over_time(jetson_power_vin_sys_5v0_watts{device_type="jetson_orin"}[$__range])
```

**디바이스별 에너지 합계**
```promql
sum by (hostname) (increase(jetson_power_vdd_in_watts[$__range]) * $__range_s / 3600)
```

### Workload Analysis Guide

워크로드별 에너지 절감 효율을 검증하는 방법:

**1. 워크로드 실행 전 시간 기록**
```bash
# 워크로드 시작 시간
START_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "Workload started at: $START_TIME"
```

**2. 워크로드 실행**
```bash
# 예시: AI 추론 작업
python inference.py
```

**3. 워크로드 종료 후 시간 기록**
```bash
# 워크로드 종료 시간
END_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "Workload ended at: $END_TIME"
```

**4. Grafana에서 분석**

- 대시보드 상단 Time Range에서 Custom Range 선택
- From: `$START_TIME`, To: `$END_TIME` 입력
- 또는 Absolute time range로 직접 지정

**5. 에너지 사용량 확인**

- "총 에너지 사용량" 패널에서 해당 구간의 에너지 확인 (Wh)
- "평균 전력" 패널에서 평균 전력 확인 (W)
- "최대 전력" 패널에서 피크 전력 확인 (W)

**6. 워크로드 비교**

예시: 두 가지 AI 모델의 에너지 효율 비교
```bash
# Model A 실행 (10:00:00 ~ 10:05:00)
# Grafana에서 확인: 5.2 Wh, 평균 3.8W

# Model B 실행 (10:10:00 ~ 10:15:00)
# Grafana에서 확인: 4.1 Wh, 평균 3.2W

# 결론: Model B가 21% 더 에너지 효율적
```

### Tips

**자동 새로고침 설정**
- 대시보드 우측 상단의 Refresh 주기 설정: `5s`, `10s`, `30s` 등
- 실시간 모니터링 시 유용

**시간 범위 단축키**
- `t z`: Zoom out time range
- `Ctrl + Z`: Zoom to data
- 드래그로 특정 구간 선택 가능

**Annotation 추가**
- 워크로드 시작/종료 시점에 Annotation 추가
- 대시보드 설정 → Annotations → `+Add annotation query`
- 수동으로 마커 추가 가능

**Dashboard Export**
```bash
# Dashboard를 JSON으로 저장 (백업용)
# Grafana UI에서: Dashboard settings → JSON Model → Copy to Clipboard
```

### Troubleshooting

**메트릭이 표시되지 않는 경우**

1. Prometheus가 메트릭을 수집하고 있는지 확인
```bash
# Prometheus Targets 확인
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090
# http://localhost:9090/targets 열기
```

2. ServiceMonitor가 정상 동작하는지 확인
```bash
kubectl get servicemonitor -n monitoring edge-devices
kubectl describe servicemonitor -n monitoring edge-devices
```

3. 디바이스가 healthy 상태인지 확인
```bash
curl http://edge-metrics-server:8081/devices
```

**"No data" 에러**

- 선택한 시간 범위에 데이터가 없을 수 있음
- 디바이스 변수 (`$device_type`, `$hostname`)가 올바른지 확인
- Prometheus 데이터소스가 올바르게 설정되었는지 확인

**에너지 계산 값이 이상한 경우**

- `$__range_s` 변수는 선택한 시간 범위(초)를 의미
- 에너지(Wh) = 전력(W) × 시간(h)
- PromQL의 `increase()` 함수는 Counter 메트릭에만 사용 가능
- Gauge 메트릭의 경우 `avg_over_time() * time_range` 사용

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
