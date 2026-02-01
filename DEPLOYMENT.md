# 🚀 Turtle Deployment Guide

Complete guide for deploying Turtle to production environments.

---

## 📋 Table of Contents

- [Environment Setup](#environment-setup)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Production Checklist](#production-checklist)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## 🔧 Environment Setup

### **Environment Variables**

Create `.env` file:

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_ENV=production
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# Database
DB_HOST=your-db-host.com
DB_PORT=5432
DB_USER=turtle_prod
DB_PASSWORD=secure_password_here
DB_NAME=turtle_production
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis
REDIS_HOST=your-redis-host.com
REDIS_PORT=6379
REDIS_PASSWORD=redis_secure_password
REDIS_DB=0
REDIS_MAX_RETRIES=3

# JWT
JWT_SECRET=your-256-bit-secret-key-here-change-this
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=1440h  # 60 days

# OTP
OTP_ENABLED=true
OTP_EXPIRY=15m
SMS_PROVIDER=twilio  # or your provider
SMS_API_KEY=your_sms_api_key

# CORS
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
CORS_ALLOWED_METHODS=GET,POST,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_OTP_REQUESTS=3
RATE_LIMIT_OTP_WINDOW=3600  # 1 hour
RATE_LIMIT_VERIFY_OTP=5
RATE_LIMIT_VERIFY_WINDOW=900  # 15 min

# Logging
LOG_LEVEL=info  # debug, info, warn, error
LOG_FORMAT=json  # json or text
```

---

## 🐳 Docker Deployment

### **1. Build Docker Image**

**Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
CMD ["./main"]
```

**Build:**
```bash
docker build -t turtle:latest .
```

---

### **2. Docker Compose (Production)**

**docker-compose.prod.yml:**
```yaml
version: '3.8'

services:
  app:
    image: turtle:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - SERVER_ENV=production
    env_file:
      - .env.production
    depends_on:
      - postgres
      - redis
    networks:
      - turtle-network
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgis/postgis:15-3.4
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - turtle-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - redis_data:/data
    networks:
      - turtle-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - app
    networks:
      - turtle-network

volumes:
  postgres_data:
  redis_data:

networks:
  turtle-network:
    driver: bridge
```

**Deploy:**
```bash
docker-compose -f docker-compose.prod.yml up -d
```

---

### **3. Nginx Configuration**

**nginx.conf:**
```nginx
events {
    worker_connections 1024;
}

http {
    upstream turtle_backend {
        server app:8080;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

    server {
        listen 80;
        server_name api.yourdomain.com;
        
        # Redirect to HTTPS
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name api.yourdomain.com;

        # SSL
        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;

        # Security headers
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;
        add_header Strict-Transport-Security "max-age=31536000" always;

        # GraphQL endpoint
        location /graphql {
            limit_req zone=api_limit burst=20 nodelay;
            
            proxy_pass http://turtle_backend/graphql;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_cache_bypass $http_upgrade;
            
            # Timeouts
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }

        # Health check
        location /health {
            proxy_pass http://turtle_backend/health;
        }

        # WebSocket for subscriptions
        location /subscriptions {
            proxy_pass http://turtle_backend/subscriptions;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "Upgrade";
            proxy_set_header Host $host;
        }
    }
}
```

---

## ☸️ Kubernetes Deployment

### **1. Kubernetes Manifests**

**deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: turtle-api
  labels:
    app: turtle
spec:
  replicas: 3
  selector:
    matchLabels:
      app: turtle
  template:
    metadata:
      labels:
        app: turtle
    spec:
      containers:
      - name: turtle
        image: your-registry/turtle:latest
        ports:
        - containerPort: 8080
        env:
        - name: SERVER_ENV
          value: "production"
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: turtle-secrets
              key: db-host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: turtle-secrets
              key: db-password
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: turtle-service
spec:
  selector:
    app: turtle
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: turtle-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: turtle-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

**secrets.yaml:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: turtle-secrets
type: Opaque
stringData:
  db-host: "your-db-host"
  db-password: "your-db-password"
  jwt-secret: "your-jwt-secret"
  redis-password: "your-redis-password"
```

**Deploy:**
```bash
kubectl apply -f secrets.yaml
kubectl apply -f deployment.yaml
```

---

## ✅ Production Checklist

### **Security**

- [ ] HTTPS enabled (SSL/TLS)
- [ ] Strong JWT secret (256+ bits)
- [ ] Database passwords rotated
- [ ] API rate limiting enabled
- [ ] CORS properly configured
- [ ] SQL injection prevention verified
- [ ] XSS protection headers set
- [ ] Environment variables secured
- [ ] Secrets management in place
- [ ] Regular security audits

### **Performance**

- [ ] Database indexes created
- [ ] Connection pooling configured
- [ ] DataLoader implemented
- [ ] Redis caching enabled
- [ ] Static asset CDN
- [ ] Gzip compression
- [ ] Query optimization
- [ ] Load testing completed

### **Reliability**

- [ ] Health checks configured
- [ ] Auto-scaling enabled
- [ ] Database backups automated
- [ ] Disaster recovery plan
- [ ] Circuit breakers implemented
- [ ] Graceful shutdown
- [ ] Zero-downtime deployments
- [ ] Rollback strategy

### **Monitoring**

- [ ] Application logs centralized
- [ ] Error tracking (Sentry)
- [ ] Performance monitoring (New Relic/DataDog)
- [ ] Uptime monitoring
- [ ] Alert system configured
- [ ] Metrics dashboard
- [ ] Database monitoring
- [ ] Redis monitoring

### **Documentation**

- [ ] API documentation updated
- [ ] Deployment runbook
- [ ] Incident response plan
- [ ] Architecture diagrams
- [ ] Change log maintained

---

## 📊 Monitoring

### **Health Endpoint**

```bash
curl https://api.yourdomain.com/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-02-01T14:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

### **Metrics to Track**

| Metric | Target | Alert Threshold |
|--------|--------|----------------|
| Response Time | < 100ms | > 500ms |
| Error Rate | < 0.1% | > 1% |
| CPU Usage | < 70% | > 85% |
| Memory Usage | < 70% | > 85% |
| DB Connections | < 20 | > 23 |
| Request Rate | Variable | Spike > 200% |

### **Logging**

**Structured JSON logging:**
```json
{
  "level": "info",
  "timestamp": "2024-02-01T14:30:00Z",
  "operation": "requestOTP",
  "user_id": "user_123",
  "duration_ms": 45,
  "status": 200
}
```

---

## 🔥 Troubleshooting

### **Common Issues**

#### **1. Database Connection Errors**

```bash
# Check connection
psql -h $DB_HOST -U $DB_USER -d $DB_NAME

# Check pool stats
SELECT * FROM pg_stat_activity;
```

#### **2. Redis Connection Issues**

```bash
# Test connection
redis-cli -h $REDIS_HOST -p $REDIS_PORT ping

# Check memory
redis-cli info memory
```

#### **3. High CPU Usage**

```bash
# Check goroutines
curl http://localhost:8080/debug/pprof/goroutine

# Profile CPU
go tool pprof http://localhost:8080/debug/pprof/profile
```

#### **4. Memory Leaks**

```bash
# Heap profile
go tool pprof http://localhost:8080/debug/pprof/heap
```

---

## 🔄 Deployment Process

### **1. Pre-Deployment**

```bash
# Run tests
go test ./...

# Build
go build -o turtle

# Smoke test
./turtle --config=staging.yaml
```

### **2. Deployment**

```bash
# Tag release
git tag v1.0.0
git push origin v1.0.0

# Build image
docker build -t turtle:v1.0.0 .
docker push your-registry/turtle:v1.0.0

# Deploy to Kubernetes
kubectl set image deployment/turtle-api turtle=your-registry/turtle:v1.0.0

# Watch rollout
kubectl rollout status deployment/turtle-api
```

### **3. Post-Deployment**

```bash
# Verify health
curl https://api.yourdomain.com/health

# Check logs
kubectl logs -f deployment/turtle-api

# Monitor metrics
# Check dashboard for errors/performance
```

### **4. Rollback (if needed)**

```bash
kubectl rollout undo deployment/turtle-api
```

---

## 📈 Scaling

### **Horizontal Scaling**

```bash
# Manual scale
kubectl scale deployment/turtle-api --replicas=5

# Auto-scaling already configured in HPA
```

### **Database Scaling**

**Read Replicas:**
```yaml
DB_PRIMARY_HOST=primary.db.com
DB_REPLICA_1=replica1.db.com
DB_REPLICA_2=replica2.db.com
```

**Connection pooling:**
```go
db.SetMaxOpenConns(25 * numberOfReplicas)
```

---

## 🔐 SSL/TLS Setup

### **Let's Encrypt (Free)**

```bash
# Install certbot
apt-get install certbot python3-certbot-nginx

# Get certificate
certbot --nginx -d api.yourdomain.com

# Auto-renewal
certbot renew --dry-run
```

---

## 📊 Performance Tuning

### **Database**

```sql
-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com';

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_addresses_user_id ON addresses(user_id);
```

### **Redis**

```bash
# Optimize memory
redis-cli CONFIG SET maxmemory 2gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

### **Go Application**

```bash
# Profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
go tool pprof http://localhost:8080/debug/pprof/heap
```

---

## 🎯 Next Steps After Deployment

1. Monitor metrics for 24 hours
2. Set up alerts
3. Document any issues
4. Plan next release
5. Gather user feedback

---

**Production deployment complete! Your API is now serving millions! 🚀**
