package main

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"YoBFF/internal/admin"
	"YoBFF/internal/app"
	"YoBFF/internal/config"
	"YoBFF/internal/gateway"
	"YoBFF/internal/logging"
	"YoBFF/internal/store"
	"YoBFF/internal/syncer"
	"YoBFF/internal/tlsutil"

	"go.uber.org/zap"
)

//go:embed web/ui/* web/ui/assets/* web/ui/vite.svg
var embeddedUI embed.FS

type serverStarter interface {
	ListenAndServe(srv *http.Server) error
	ListenAndServeTLS(srv *http.Server) error
}

type defaultServerStarter struct{}

// ListenAndServe 启动 HTTP 服务监听。
// 参数：srv 为 HTTP 服务对象。
// 返回：服务退出错误。
// 异常：无。
func (defaultServerStarter) ListenAndServe(srv *http.Server) error {
	return srv.ListenAndServe()
}

// ListenAndServeTLS 启动 HTTPS 服务监听。
// 参数：srv 为 HTTPS 服务对象。
// 返回：服务退出错误。
// 异常：无。
func (defaultServerStarter) ListenAndServeTLS(srv *http.Server) error {
	return srv.ListenAndServeTLS("", "")
}

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

	configPath := config.EnvOrDefault("CONFIG_PATH", "config/config.json")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err = run(ctx, logger, logRuntime, configPath, defaultServerStarter{}); err != nil {
		logger.Fatal("服务启动失败", zap.Error(err))
	}
}

// run 负责执行可测试的启动流程。
// 参数：ctx 为退出信号上下文，logger 为日志实例，logRuntime 为日志运行时，configPath 为配置文件路径，starter 为服务启动器。
// 返回：启动或初始化失败错误。
// 异常：无。
func run(
	ctx context.Context,
	logger *zap.Logger,
	logRuntime *logging.Runtime,
	configPath string,
	starter serverStarter,
) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if logRuntime == nil {
		return fmt.Errorf("日志运行时为空")
	}
	if configPath == "" {
		return fmt.Errorf("配置路径为空")
	}
	if starter == nil {
		starter = defaultServerStarter{}
	}

	manager, err := config.NewManager(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	finalConfig := config.ApplyEnvOverrides(manager.CurrentConfig())
	if err = manager.Apply(finalConfig); err != nil {
		return fmt.Errorf("应用环境变量覆盖配置失败: %w", err)
	}

	cdnSyncer := syncer.NewService(manager, logger)
	cdnSyncer.Start(ctx)

	logPipeline := logging.NewPipeline(4096, logger)
	defer logPipeline.Close()

	dataHandler := gateway.NewHandler(manager, logPipeline)
	dbPath := config.EnvOrDefault("CONFIG_DB_PATH", "config/config.db")
	configStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("初始化配置存储失败: %w", err)
	}
	adminSrv := admin.NewServer(manager, logRuntime, logPipeline, configStore)
	uiHandler, err := buildUIHandler()
	if err != nil {
		return fmt.Errorf("初始化管理端静态资源失败: %w", err)
	}
	uiHandler = adminSrv.WrapHandler(uiHandler)
	rootHandler := app.BuildRootHandler(dataHandler, adminSrv.Handler(), uiHandler)

	reloadInterval := config.EnvDurationSeconds("CONFIG_RELOAD_INTERVAL_SECONDS", 3)
	watcher, err := config.NewWatcher(manager, logger, reloadInterval)
	if err != nil {
		return fmt.Errorf("初始化配置监听失败: %w", err)
	}
	watcher.Start(ctx)

	cfg := manager.CurrentConfig()
	if cfg.ControlPlane.AdminListenAddr != "" {
		logger.Info(
			"注意：AdminListenAddr 配置已废弃，管理接口已合并至主端口 /admin 路径",
			zap.String("ignored_addr", cfg.ControlPlane.AdminListenAddr),
		)
	}

	httpSrv := &http.Server{
		Addr:    cfg.DataPlane.HTTPListenAddr,
		Handler: rootHandler,
	}

	errCh := make(chan error, 3)

	go func() {
		logger.Info("HTTP 服务启动", zap.String("listen", httpSrv.Addr))
		if serveErr := starter.ListenAndServe(httpSrv); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	var httpsSrv *http.Server
	if cfg.DataPlane.EnableHTTPS && cfg.DataPlane.HTTPSListenAddr != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

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
			httpsHandler := app.WithHSTS(rootHandler, manager)
			httpsSrv = &http.Server{
				Addr:      cfg.DataPlane.HTTPSListenAddr,
				Handler:   httpsHandler,
				TLSConfig: tlsConfig,
			}

			httpSrv.Handler = app.RedirectHTTPToHTTPS(httpSrv.Handler, cfg.DataPlane.HTTPSListenAddr)

			go func() {
				logger.Info("HTTPS 服务启动", zap.String("listen", httpsSrv.Addr))
				if serveErr := starter.ListenAndServeTLS(httpsSrv); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
					errCh <- serveErr
				}
			}()
		}
	} else if !cfg.DataPlane.EnableHTTPS {
		logger.Warn("HTTPS 已禁用，已跳过 HTTPS 服务")
	} else {
		logger.Warn("HTTPS 监听地址为空，已跳过 HTTPS 服务")
	}

	select {
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅关闭")
	case serveErr := <-errCh:
		logger.Error("服务异常退出", zap.Error(serveErr))
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		config.EnvDurationSeconds("SHUTDOWN_TIMEOUT_SECONDS", 5),
	)
	defer shutdownCancel()

	shutdownServer(shutdownCtx, logger, "HTTP 服务", httpSrv)
	if httpsSrv != nil {
		shutdownServer(shutdownCtx, logger, "HTTPS 服务", httpsSrv)
	}
	return nil
}

// buildUIHandler 选择外部目录或内嵌资源初始化管理端静态资源处理器。
// 参数：无。
// 返回：管理端静态资源处理器与初始化错误。
// 异常：资源缺失或创建失败时返回错误。
func buildUIHandler() (http.Handler, error) {
	dir := strings.TrimSpace(os.Getenv("ADMIN_UI_DIR"))
	if dir != "" {
		return app.NewUIHandler(os.DirFS(dir))
	}
	sub, err := fs.Sub(embeddedUI, "web/ui")
	if err != nil {
		return nil, err
	}
	return app.NewUIHandler(sub)
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
