package handler

import (
	"encoding/json"
	"net/http"
	"time"
	"yunyuyuan/anypaste/internal/auth"
)

// RegisterLoginHandler 挂载登录入口。登录本身不能走鉴权拦截器（此时还没有 token）。
// 路由用 /login（不带 /api 前缀）：dev 代理会把前端 /api/login 的 /api 剥掉再转发，
// 与 /file/* 等其它后端路由保持一致。
func RegisterLoginHandler(mux *http.ServeMux) {
	mux.HandleFunc("/login", loginHandler)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := auth.VerifyPasswd(body.Password); err != nil {
		// 轻微延迟，钝化暴力破解
		time.Sleep(300 * time.Millisecond)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := auth.IssueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}
