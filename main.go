package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"YoBFF/internal/admin"
	"YoBFF/internal/app"
	"YoBFF/internal/config"
	"YoBFF/internal/gateway"
	"YoBFF/internal/logging"
	"YoBFF/internal/tlsutil"

	"go.uber.org/zap"
)

// main 负责启动控制平面、数据平面与配置热重载流程。
// 参数：无。
// 返回：无。
// 异常：关键初始化失败时记录错误并退出进程。
func main() {
	// 加载环境变量
	if err := config.LoadDotEnv(".env"); err != nil {
		panic(err)
	}

	// 初始化日志系统
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		panic(err)
	}
	defer logRuntime.Sync()
	logger := logRuntime.Logger()

	// 加载初始配置
	configPath := config.EnvOrDefault("CONFIG_PATH", "config/config.json")
	manager, err := config.NewManager(configPath)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 应用环境变量覆盖
	finalConfig := config.ApplyEnvOverrides(manager.CurrentConfig())
	if err = manager.Apply(finalConfig); err != nil {
		logger.Fatal("应用环境变量覆盖配置失败", zap.Error(err))
	}

	// 初始化异步日志管线
	logPipeline := logging.NewPipeline(4096, logger)
	defer logPipeline.Close()

	// 初始化业务处理器
	dataHandler := gateway.NewHandler(manager, logPipeline)
	adminSrv := admin.NewServer(manager, logRuntime, logPipeline)

	rootHandler := app.BuildRootHandler(dataHandler, adminSrv.Handler())

	// 初始化配置监听器，支持热重载
	reloadInterval := config.EnvDurationSeconds("CONFIG_RELOAD_INTERVAL_SECONDS", 3)
	watcher, err := config.NewWatcher(manager, logger, reloadInterval)
	if err != nil {
		logger.Fatal("初始化配置监听失败", zap.Error(err))
	}

	// 监听系统信号，实现优雅退出
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	watcher.Start(ctx)

	cfg := manager.CurrentConfig()

	// 提示用户 Admin 端口合并信息
	if cfg.ControlPlane.AdminListenAddr != "" {
		logger.Info("注意：AdminListenAddr 配置已废弃，管理接口已合并至主端口 /admin 路径",
			zap.String("ignored_addr", cfg.ControlPlane.AdminListenAddr))
	}

	// 初始化 HTTP 服务器
	httpSrv := &http.Server{
		Addr:    cfg.DataPlane.HTTPListenAddr,
		Handler: rootHandler,
	}

	errCh := make(chan error, 3)

	// 启动 HTTP 服务
	go func() {
		logger.Info("HTTP 服务启动", zap.String("listen", httpSrv.Addr))
		if serveErr := httpSrv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	// 初始化 HTTPS 服务器（如果启用）
	var httpsSrv *http.Server
	if cfg.DataPlane.EnableHTTPS && cfg.DataPlane.HTTPSListenAddr != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// 证书加载逻辑
		if manager.HasCertificates() {
			tlsConfig.GetCertificate = manager.GetCertificate
			logger.Info("HTTPS 启用配置证书模式")
		} else {
			autoCert, mode, certErr := tlsutil.BuildCertificate(cfg)
			if certErr != nil {
				logger.Error("自动证书生成失败，HTTPS 不可用", zap.Error(certErr))
			} else {
				tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
					return autoCert, nil
				}
				logger.Warn("HTTPS 启用自动证书模式", zap.String("mode", mode))
			}
		}

		if tlsConfig.GetCertificate == nil {
			logger.Warn("未检测到证书且自动签发失败，已跳过 HTTPS 服务")
		} else {
			// 为 HTTPS 处理器添加 HSTS 支持
			httpsHandler := app.WithHSTS(rootHandler, manager)

			httpsSrv = &http.Server{
				Addr:      cfg.DataPlane.HTTPSListenAddr,
				Handler:   httpsHandler,
				TLSConfig: tlsConfig,
			}

			// 如果启用 HTTPS，将 HTTP 服务处理器替换为强制重定向
			// 这将覆盖之前的 rootHandler，确保所有 HTTP 流量跳转至 HTTPS
			httpSrv.Handler = app.RedirectHTTPToHTTPS(httpSrv.Handler, cfg.DataPlane.HTTPSListenAddr)

			// 启动 HTTPS 服务
			go func() {
				logger.Info("HTTPS 服务启动", zap.String("listen", httpsSrv.Addr))
				if serveErr := httpsSrv.ListenAndServeTLS("", ""); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
					errCh <- serveErr
				}
			}()
		}
	} else if !cfg.DataPlane.EnableHTTPS {
		logger.Warn("HTTPS 已禁用，已跳过 HTTPS 服务")
	} else {
		logger.Warn("HTTPS 监听地址为空，已跳过 HTTPS 服务")
	}

	// 等待退出信号或服务错误
	select {
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅关闭")
	case serveErr := <-errCh:
		logger.Error("服务异常退出", zap.Error(serveErr))
	}

	// 执行优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), config.EnvDurationSeconds("SHUTDOWN_TIMEOUT_SECONDS", 5))
	defer shutdownCancel()

	shutdownServer(shutdownCtx, logger, "HTTP 服务", httpSrv)
	if httpsSrv != nil {
		shutdownServer(shutdownCtx, logger, "HTTPS 服务", httpsSrv)
	}
}

// shutdownServer 统一执行服务优雅关闭并记录结果日志。
// 参数：ctx 为关闭超时上下文，logger 为日志实例，name 为服务名称，srv 为服务对象。
// 返回：无。
// 异常：关闭失败时记录错误日志。
func shutdownServer(ctx context.Context, logger *zap.Logger, name string, srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务关闭失败", zap.String("server", name), zap.Error(err))
		return
	}
	logger.Info("服务关闭完成", zap.String("server", name))
}
