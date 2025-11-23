# 镜像名和默认 tag
IMAGE_NAME := 93maoshui/assistant-gateway-admin
IMAGE_TAG  ?= latest
COMPOSE_PROJECT_NAME ?= assistant

# 本地构建
go-build:
	go build -o admin ./cmd/admin

# 本地运行
go-run: go-build
	./admin

# 本地构建镜像（用于 docker-compose / K8s 部署）
build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# 构建并推送到 Docker Hub
push: build
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

# 按时间生成一个 dev tag 并推送（可选）
push-dev:
	IMAGE_TAG=dev-$(shell date +%Y%m%d-%H%M%S) $(MAKE) push

# 本地启动（使用 docker-compose）
run:
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --build

# 查看日志
logs:
	docker compose -p $(COMPOSE_PROJECT_NAME) logs -f

# 停止容器
stop:
	docker compose -p $(COMPOSE_PROJECT_NAME) stop

# 删除容器（保留卷）
down:
	docker compose -p $(COMPOSE_PROJECT_NAME) down

.PHONY: go-build go-run build push push-dev run logs stop down

