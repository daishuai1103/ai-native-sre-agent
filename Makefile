.PHONY: all build backend frontend clean docker-up docker-down dev-backend dev-frontend

all: build

# 编译后端
backend:
	cd backend && go build -o bin/server ./cmd/server

# 编译前端
frontend:
	cd frontend && npm install && npm run build

# 全部编译
build: backend frontend
	@echo "✅ 编译完成"
	@echo "后端: backend/bin/server"
	@echo "前端: frontend/dist/"

# 启动 Docker
docker-up:
	docker compose up -d

# 停止 Docker
docker-down:
	docker compose down

# 开发模式后端
dev-backend:
	cd backend && go run ./cmd/server

# 开发模式前端
dev-frontend:
	cd frontend && npm run dev

# 清理
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	@echo "🧹 清理完成"
