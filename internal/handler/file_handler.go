package handler

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"yunyuyuan/anypaste/internal/auth"
	"yunyuyuan/anypaste/internal/service"
)

// UploadDir is the directory where uploaded files are stored. main() may
// override it from the UPLOAD_DIR env var before the server starts serving.
var UploadDir = "uploads"

type FileHandler struct {
	pasteSvc *service.PasteService
}

func RegisterFileHandler(mux *http.ServeMux, svc *service.PasteService) {
	h := &FileHandler{pasteSvc: svc}

	// 上传需要鉴权；下载只凭 id 即可（id 即凭证）
	mux.Handle("/file/upload/{id}", auth.JWTMiddleware(http.HandlerFunc(h.UploadHandler)))
	mux.HandleFunc("/file/download/{id}", h.DownloadHandler)
}

func (h *FileHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pasteId := r.PathValue("id")

	pasteItem, err := h.pasteSvc.GetPaste(r.Context(), pasteId)
	if pasteItem == nil || err != nil {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 只保存第一个文件，不支持多文件上传
	savedName, err := saveFirstFile(reader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if savedName == "" {
		http.Error(w, "no file provided", http.StatusBadRequest)
		return
	}

	if err := h.pasteSvc.UpdatePasteFileName(r.Context(), pasteId, savedName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func saveFirstFile(reader *multipart.Reader) (string, error) {
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if part.FormName() != "file" {
			err := part.Close()
			if err != nil {
				return "", err
			}
			continue
		}

		ext := filepath.Ext(filepath.Base(part.FileName()))
		savedName := fmt.Sprintf("file_%d%s", time.Now().UnixMicro(), ext)

		dst, err := os.Create(parseFilePath(savedName))
		if err != nil {
			err := part.Close()
			if err != nil {
				return "", err
			}
			return "", err
		}

		// 使用固定缓冲区拷贝，避免一次性读入内存
		buf := make([]byte, 64*1024)
		_, copyErr := io.CopyBuffer(dst, part, buf)
		// 显式关闭：Close 会刷新缓冲并可能返回写入错误，必须检查
		closeErr := dst.Close()
		err = part.Close()
		if err != nil {
			return "", err
		}
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return savedName, nil
	}
}

func (h *FileHandler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	pasteId := r.PathValue("id")
	pasteItem, err := h.pasteSvc.GetPaste(r.Context(), pasteId)
	if pasteItem == nil || err != nil {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}
	filename := pasteItem.FileName
	if filename == nil || *filename == "" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	file, err := os.Open(parseFilePath(*filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 交给 http.ServeContent：自动处理 Range 请求（支持断点续传/重试）、
	// Content-Length、Accept-Ranges 以及条件请求
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", *filename))
	http.ServeContent(w, r, *filename, stat.ModTime(), file)
}

func parseFilePath(filename string) string {
	return filepath.Join(UploadDir, filepath.Base(filename))
}
