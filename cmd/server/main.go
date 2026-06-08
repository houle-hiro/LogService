package main

import (
	"fmt"
	"log"
	"log-service/internal/api"
	"log-service/internal/config"
	"log-service/internal/service"
	"net/http"
)

func main() {
	// 加载配置文件
	cfg, err := config.LoadDefault()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("配置加载成功: 端口=%d, 存储路径=%s, 保留时间=%d秒",
		cfg.Server.Port, cfg.Storage.Path, cfg.Storage.MaxAgeSeconds)

	// 初始化存储
	storage, err := service.NewLocalStorage(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	// 启动老化清理
	cleanup := service.NewCleanup(storage, cfg.Storage.MaxAgeSeconds)
	cleanup.Start()
	defer cleanup.Stop()

	// 创建 HTTP 处理器
	handler := api.NewHandler(storage, cleanup, cfg)

	// 注册路由
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 应用中间件
	var h http.Handler = mux
	for _, mw := range handler.Middleware() {
		h = mw(h)
	}

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      h,
		ReadTimeout:  cfg.GetReadTimeout(),
		WriteTimeout: cfg.GetWriteTimeout(),
		IdleTimeout:  cfg.GetIdleTimeout(),
	}

	log.Printf("日志服务启动，监听端口 %d", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
