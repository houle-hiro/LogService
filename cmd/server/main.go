package main

import (
	"fmt"
	"log"
	"net/http"

	"log-service/internal/api"
	"log-service/internal/config"
	"log-service/internal/service"
)

func main() {
	cfg, err := config.LoadDefault()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("配置加载成功: 端口=%d, 存储路径=%s, 保留时长=%s, 鉴权=%v, 最大上传=%dMB",
		cfg.Server.Port, cfg.Storage.Path, cfg.GetMaxAge(), cfg.AuthEnabled(), cfg.Upload.MaxSizeMB)

	storage, err := service.NewLocalStorage(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	cleanup := service.NewCleanup(storage, cfg.GetMaxAgeSeconds())
	cleanup.Start()
	defer cleanup.Stop()

	handler := api.NewHandler(storage, cleanup, cfg)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	var h http.Handler = mux
	for _, mw := range handler.Middleware() {
		h = mw(h)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      h,
		ReadTimeout:  cfg.GetReadTimeout(),
		WriteTimeout: cfg.GetWriteTimeout(),
		IdleTimeout:  cfg.GetIdleTimeout(),
	}

	log.Printf("文件中转服务启动，监听端口 %d", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}