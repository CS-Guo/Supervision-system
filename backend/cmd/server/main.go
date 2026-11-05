// Command server 是服务运行入口，只负责组装基础设施依赖；
// 具体应用行为保留在 internal 各包中。
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"supervision-system/internal/config"
	"supervision-system/internal/httpapi"
	mysqlrepo "supervision-system/internal/repository/mysql"
	"supervision-system/internal/service"
	"supervision-system/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, e := config.Load()
	if e != nil {
		log.Error("load config", "error", e)
		os.Exit(1)
	}
	db, e := mysqlrepo.Open(cfg.DSN)
	if e != nil {
		log.Error("open database", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	repo := mysqlrepo.New(db)
	svc := service.New(repo, storage.Local{Root: cfg.UploadDir}, cfg.SessionTTL, cfg.MaxUploadBytes)
	// 显式设置超时和请求头限制，避免慢速或已断开的客户端长期占用资源。
	server := &http.Server{Addr: cfg.Addr, Handler: httpapi.New(svc, log, cfg.FrontendDir, cfg.MaxUploadBytes).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		log.Info("server started", "addr", cfg.Addr)
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Error("server stopped", "error", e)
			os.Exit(1)
		}
	}()
	// 进程收到终止信号后停止接收新请求，并等待正在处理的请求完成。
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e := server.Shutdown(ctx); e != nil {
		log.Error("graceful shutdown", "error", e)
	}
}
