# VSITE 构建与运维脚本
GO ?= $(HOME)/go/bin/go1.25.0

.PHONY: build run init-db seed test-api clean

build:
	$(GO) build -o bin/vsite .

# 直接运行（首次会自动建表并写入默认管理员 ADMIN/123456）
run: build
	./bin/vsite -addr :8080

# 初始化数据库与账号（需 root 权限；只需执行一次）
init-db:
	sudo mysql < scripts/init_db.sql

# 写入示例视频/博主数据
seed: build
	./bin/vsite -seed

# 接口端到端测试（需先启动服务）
test-api:
	bash scripts/test_api.sh http://127.0.0.1:8080

clean:
	rm -rf bin
