package main

import (
	"log"
	"net/http"
	"os"
	"yunyuyuan/anypaste/gen/paste/v1/pastev1connect"
	"yunyuyuan/anypaste/internal/auth"
	"yunyuyuan/anypaste/internal/handler"
	"yunyuyuan/anypaste/internal/model"
	"yunyuyuan/anypaste/internal/service"
	"yunyuyuan/anypaste/internal/utils"
	"yunyuyuan/anypaste/internal/web"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/joho/godotenv"
)

func main() {
	// 开发时加载 .env.local；文件不存在不报错（生产用真实环境变量）。
	_ = godotenv.Load(".env.local")

	// 缺关键密钥时只告警、不阻止启动（方便首次试跑），但功能会受影响。
	for _, e := range []struct{ key, impact string }{
		{"APP_PASSWD", "login will always fail"},
		{"JWT_SECRET", "tokens are signed with an empty secret (insecure)"},
	} {
		if os.Getenv(e.key) == "" {
			log.Printf("WARNING: %s is not set — %s", e.key, e.impact)
		}
	}

	// 数据位置可通过环境变量覆盖，便于容器里挂卷持久化。
	handler.UploadDir = utils.EnvOr("UPLOAD_DIR", "uploads")
	if err := os.MkdirAll(handler.UploadDir, 0o755); err != nil {
		log.Fatalf("create upload dir: %v", err)
	}

	db, err := model.Open(utils.EnvOr("DB_PATH", "data.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	// 启动即迁移，保证全新容器开箱即用（幂等）。
	if err := db.AutoMigrate(&model.Paste{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	pasteService := service.NewPasteService(model.NewPasteRepo(db))

	// 所有后端接口都挂在 apiMux 上（根相对路径）。
	apiMux := http.NewServeMux()
	apiMux.Handle(pastev1connect.NewPasteServiceHandler(
		handler.NewPasteHandler(pasteService),
		// Validation via Protovalidate is almost always recommended
		connect.WithInterceptors(validate.NewInterceptor(), auth.NewAuthUnaryInterceptor()),
	))
	handler.RegisterFileHandler(apiMux, pasteService)
	handler.RegisterLoginHandler(apiMux)

	// 根路由：/api/* 去掉前缀后交给 apiMux，其余交给内嵌前端（SPA，带 history fallback）。
	spa, err := web.Handler()
	if err != nil {
		log.Fatalf("init web: %v", err)
	}
	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", apiMux))
	// 纯文本 CLI 指南，给 `curl <host>/help` 用（根路径，不在 /api 下）
	handler.RegisterHelpHandler(root)
	// CLI 二进制挂在根 /cli/（公开、干净的 curl 下载地址，与 web 的 /api 解耦）
	handler.RegisterCLIHandler(root)
	root.Handle("/", spa)

	addr := utils.EnvOr("ADDR", ":8080")
	p := new(http.Protocols)
	p.SetHTTP1(true)
	// Use h2c so we can serve HTTP/2 without TLS.
	p.SetUnencryptedHTTP2(true)
	s := http.Server{
		Addr:      addr,
		Handler:   root,
		Protocols: p,
	}
	log.Printf("listening on %s", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
