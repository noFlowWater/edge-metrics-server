# Edge Metrics Server

중앙 집중식 엣지 디바이스 설정 관리 서버 + Kubernetes 통합

## Features

- 엣지 디바이스 설정 관리 (CRUD)
- 디바이스 상태 모니터링
- 설정 변경 시 자동 리로드 트리거
- **Kubernetes 통합**: 외부 엣지 디바이스를 Prometheus가 스크래핑할 수 있도록 Service/Endpoints로 가상화

## Requirements

- Go 1.23.4+
- SQLite3
- Kubernetes cluster (Kubernetes 기능 사용 시)

## Quick Start

### Local 실행

```bash
# 빌드
go build -o edge-metrics-server

# 실행
./edge-metrics-server

# 또는 환경변수 지정
PORT=8080 DB_PATH=/data/config.db ./edge-metrics-server
```

### Kubernetes 배포

#### 1. RBAC 리소스 생성 (최초 1회)

```bash
kubectl apply -f manifests/rbac.yaml
```

이 명령은 다음을 생성합니다:
- ServiceAccount: `edge-metrics-server`
- Role: `edge-metrics-manager` (services, endpoints 권한)
- RoleBinding: ServiceAccount와 Role 연결

#### 2. 데이터 영구 저장 설정 (선택)

SQLite DB를 영구적으로 보관하려면 PVC를 사용하세요:

```bash
# PVC 생성
kubectl apply -f manifests/pvc.yaml

# deployment.yaml에서 PVC 사용 설정
# volumes 섹션의 주석 처리된 persistentVolumeClaim 부분 활성화
```

**deployment.yaml 수정:**
```yaml
volumes:
- name: data
  # emptyDir: {}  # 이 줄 주석 처리
  persistentVolumeClaim:
    claimName: edge-metrics-data  # 이 부분 활성화
```

> **참고**: emptyDir 사용 시 Pod 재시작 시 데이터가 삭제됩니다. 프로덕션 환경에서는 PVC 사용을 권장합니다.

#### 3. 서비스 타입 설정 (선택)

**ClusterIP (기본)**: 클러스터 내부에서만 접근
```yaml
# service.yaml (기본값, 수정 불필요)
spec:
  type: ClusterIP
```

**NodePort**: 클러스터 외부에서 `<NodeIP>:30081`로 접근
```yaml
# service.yaml 수정
spec:
  # type: ClusterIP  # 이 줄 주석 처리
  type: NodePort     # 이 부분 활성화
  ports:
  - port: 8081
    targetPort: 8081
    nodePort: 30081  # 이 줄도 활성화 (선택)
```

#### 4. 서버 배포

```bash
kubectl apply -f manifests/deployment.yaml
kubectl apply -f manifests/service.yaml
```

#### 5. 배포 확인

```bash
kubectl get pods -n monitoring
kubectl logs -n monitoring deployment/edge-metrics-server
```

## Kubernetes Integration

### 개요

edge-metrics-server는 외부 엣지 디바이스(Jetson, Raspberry Pi 등)를 Kubernetes 클러스터 내부의 Service/Endpoints 리소스로 매핑하여, Prometheus가 클러스터 내부 Pod처럼 스크래핑할 수 있도록 합니다.

### 작동 방식

```
┌─────────────────────────────────────────────────┐
│  Kubernetes Cluster (monitoring namespace)      │
│                                                  │
│  ┌──────────────────────────────────────────┐  │
│  │ Prometheus                                │  │
│  │ - ServiceMonitor 자동 감지               │  │
│  │ - edge-device-* Service 스크래핑          │  │
│  └──────────────────────────────────────────┘  │
│              ↓ (scrapes)                        │
│  ┌──────────────────────────────────────────┐  │
│  │ Service: edge-device-edge-01              │  │
│  │ Endpoints: 192.168.1.10:9100             │  │
│  └──────────────────────────────────────────┘  │
│              ↓ (points to)                      │
└──────────────┼───────────────────────────────────┘
               ↓ (외부 네트워크)
     ┌─────────────────────┐
     │ 실제 엣지 디바이스    │
     │ IP: 192.168.1.10     │
     │ Exporter: :9100      │
     └─────────────────────┘
```

### API 엔드포인트

#### POST /kubernetes/sync

현재 healthy 상태인 모든 디바이스를 Kubernetes Service + Endpoints로 동기화합니다.

**요청:**
```bash
curl -X POST http://edge-metrics-server:8081/kubernetes/sync \
  -H "Content-Type: application/json" \
  -d '{"namespace": "monitoring"}'
```

**응답:**
```json
{
  "status": "synced",
  "created": [
    {"device_id": "edge-01", "service": "edge-device-edge-01", "status": "created"}
  ],
  "updated": [],
  "deleted": [],
  "failed": [],
  "total_healthy": 1
}
```

#### GET /kubernetes/manifests

Healthy 디바이스들의 Kubernetes YAML 매니페스트를 생성합니다 (수동 적용용).

**요청:**
```bash
curl http://edge-metrics-server:8081/kubernetes/manifests?namespace=monitoring > edge-devices.yaml
kubectl apply -f edge-devices.yaml
```

#### DELETE /kubernetes/cleanup

monitoring 네임스페이스의 모든 edge-device-* 리소스를 삭제합니다.

**요청:**
```bash
curl -X DELETE http://edge-metrics-server:8081/kubernetes/cleanup?namespace=monitoring
```

**응답:**
```json
{
  "status": "cleaned",
  "deleted_services": ["edge-device-edge-01", "edge-device-edge-02"],
  "deleted_endpoints": ["edge-device-edge-01", "edge-device-edge-02"],
  "namespace": "monitoring"
}
```

### Prometheus 통합

#### ServiceMonitor 설정 (Prometheus Operator 사용 시)

Prometheus Operator를 사용 중이라면 ServiceMonitor를 적용하여 자동 디스커버리를 활성화하세요:

**적용:**
```bash
kubectl apply -f manifests/servicemonitor.yaml
```

**확인:**
```bash
# ServiceMonitor 생성 확인
kubectl get servicemonitor -n monitoring

# Prometheus targets 확인 (포트포워드 후 브라우저에서 확인)
kubectl port-forward -n monitoring svc/prometheus-operated 9090:9090
# http://localhost:9090/targets 열기
```

**동작 원리:**
- ServiceMonitor는 `app=edge-exporter` 레이블을 가진 모든 Service를 자동 감지
- Prometheus가 해당 Service의 `/metrics` 엔드포인트를 30초마다 스크래핑
- 엣지 디바이스의 메트릭이 Prometheus에 자동으로 수집됨

> **참고**: ServiceMonitor는 Prometheus Operator가 설치되어 있어야 작동합니다.
> ```bash
> # Prometheus Operator 설치 (미설치 시)
> helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
> helm repo update
> helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring
> ```

### 사용 시나리오

#### 시나리오 1: 수동 동기화

```bash
# 1. 엣지 디바이스 등록
curl -X PUT http://edge-metrics-server:8081/config/edge-01 \
  -d '{"device_type": "jetson_orin", "port": 9100}'

# 2. 디바이스 상태 확인
curl http://edge-metrics-server:8081/devices

# 3. Kubernetes에 동기화
kubectl exec -n monitoring deployment/edge-metrics-server -- \
  curl -X POST http://localhost:8081/kubernetes/sync

# 4. 생성된 리소스 확인
kubectl get svc,endpoints -n monitoring -l managed_by=edge-metrics-server
```

#### 시나리오 2: 외부에서 API 호출 (port-forward)

```bash
# 1. Port forward 설정
kubectl port-forward -n monitoring svc/edge-metrics-server 8081:8081

# 2. 로컬에서 API 호출
curl -X POST http://localhost:8081/kubernetes/sync \
  -d '{"namespace": "monitoring"}'
```

#### 시나리오 3: CronJob으로 주기적 동기화

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: edge-device-sync
  namespace: monitoring
spec:
  schedule: "*/5 * * * *"  # 5분마다
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: sync
            image: curlimages/curl:latest
            args:
            - sh
            - -c
            - |
              curl -X POST http://edge-metrics-server:8081/kubernetes/sync \
                -H "Content-Type: application/json" \
                -d '{"namespace": "monitoring"}'
          restartPolicy: OnFailure
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | 서버 포트 |
| `DB_PATH` | ./config.db | SQLite 데이터베이스 경로 |
| `SERVER_URL` | http://localhost:8081 | 자기 자신의 URL (K8s sync에서 사용) |

## API Documentation

자세한 API 문서는 [API.md](./API.md)를 참조하세요.

## Architecture

```
edge-metrics-server/
├── main.go                     # 엔트리 포인트
├── database/                   # SQLite 데이터베이스
├── models/                     # 데이터 모델
├── repository/                 # 데이터베이스 CRUD
├── handlers/                   # HTTP 핸들러
│   ├── handlers.go            # 디바이스 관리 API
│   └── kubernetes_handler.go  # Kubernetes 통합 API
├── router/                     # 라우트 설정
├── kubernetes/                 # Kubernetes 클라이언트
│   ├── client.go              # K8s 클라이언트 초기화
│   ├── service.go             # Service 리소스 관리
│   ├── endpoints.go           # Endpoints 리소스 관리
│   └── sync.go                # 동기화 로직
└── manifests/                  # Kubernetes 매니페스트
    ├── rbac.yaml              # RBAC 권한
    ├── deployment.yaml        # 서버 배포
    └── service.yaml           # 서버 Service
```

## Security

### RBAC 권한

edge-metrics-server는 다음 Kubernetes 권한만 필요합니다:

- **services**: get, list, create, update, patch, delete
- **endpoints**: get, list, create, update, patch, delete
- **servicemonitors** (선택): get, list, create, update, patch, delete

### 네트워크 요구사항

Kubernetes Pod에서 외부 엣지 디바이스로 접근 가능해야 합니다:
- 방화벽에서 디바이스 포트(기본 9100) 허용
- VPN/Tailscale 등 사설망 구성 시 네트워크 라우팅 설정

## Troubleshooting

### Kubernetes client not initialized

```
Kubernetes client not initialized: failed to create Kubernetes config
```

**원인**: Pod가 ServiceAccount를 사용하지 않거나, kubeconfig가 없음

**해결**:
1. Deployment에서 `serviceAccountName: edge-metrics-server` 확인
2. RBAC 리소스가 생성되었는지 확인: `kubectl get sa,role,rolebinding -n monitoring`

### Service created but endpoints empty

```
kubectl get endpoints -n monitoring edge-device-edge-01
# No endpoints available
```

**원인**: 디바이스 IP 주소가 등록되지 않음

**해결**:
1. 디바이스 상태 확인: `curl http://server/devices`
2. IP 주소가 비어있다면 디바이스에서 서버로 설정 등록 필요

### Prometheus not scraping edge devices

**확인 사항**:
1. ServiceMonitor가 올바른 레이블 셀렉터 사용하는지 확인
2. Prometheus Operator가 해당 네임스페이스를 감시하는지 확인
3. Prometheus 로그에서 target discovery 확인

## License

MIT
