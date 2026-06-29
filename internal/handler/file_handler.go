package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"yunyuyuan/anypaste/internal/auth"
	"yunyuyuan/anypaste/internal/service"
	"yunyuyuan/anypaste/internal/uploadproto"
)

// UploadDir is the directory where uploaded files are stored. main() may
// override it from the UPLOAD_DIR env var before the server starts serving.
var UploadDir = "uploads"

// PartialDir is the subdirectory under UploadDir holding in-progress resumable
// uploads (one .part file per paste). It is a dotted dir so the orphan-file
// cleanup, which skips subdirectories, never touches a live upload; abandoned
// partials are reclaimed by cleanup.SweepPartials.
const PartialDir = ".partial"

type FileHandler struct {
	pasteSvc *service.PasteService
}

func RegisterFileHandler(mux *http.ServeMux, svc *service.PasteService) {
	h := &FileHandler{pasteSvc: svc}

	// 上传需要鉴权；下载只凭 id 即可（id 即凭证）
	mux.Handle("/file/upload/{id}", auth.JWTMiddleware(http.HandlerFunc(h.UploadHandler)))
	mux.HandleFunc("/file/download/{id}", h.DownloadHandler)
}

// UploadHandler implements the resumable chunked-upload protocol (see
// internal/uploadproto): HEAD reports how many bytes are already stored, POST
// appends one raw-body chunk at a given offset and finalizes once the file is
// complete. Chunking keeps every request small enough to clear proxies with
// tight body-size/timeout limits, and lets a dropped transfer resume.
func (h *FileHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	pasteId := r.PathValue("id")
	pasteItem, err := h.pasteSvc.GetPaste(r.Context(), pasteId)
	if pasteItem == nil || err != nil {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodHead:
		h.headOffset(w, pasteId)
	case http.MethodPost:
		h.appendChunk(w, r, pasteId)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// headOffset reports the bytes already received for this paste so the client
// knows where to (re)start. 0 means no partial exists yet.
func (h *FileHandler) headOffset(w http.ResponseWriter, pasteId string) {
	size, err := partialSize(pasteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(uploadproto.HeaderUploadOffset, strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
}

// appendChunk appends the request body to the paste's .part file at the offset
// the client claims, then finalizes if the file is now complete.
func (h *FileHandler) appendChunk(w http.ResponseWriter, r *http.Request, pasteId string) {
	offset, err := strconv.ParseInt(r.Header.Get(uploadproto.HeaderUploadOffset), 10, 64)
	if err != nil || offset < 0 {
		http.Error(w, "invalid "+uploadproto.HeaderUploadOffset, http.StatusBadRequest)
		return
	}
	total, err := strconv.ParseInt(r.Header.Get(uploadproto.HeaderUploadLength), 10, 64)
	if err != nil || total < 0 {
		http.Error(w, "invalid "+uploadproto.HeaderUploadLength, http.StatusBadRequest)
		return
	}
	if total > uploadproto.MaxFileSize {
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := os.MkdirAll(partialDirPath(), 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	partPath := partialPath(pasteId)

	// The client's offset must match what we already have; otherwise reply 409
	// with the authoritative offset so it can re-sync (idempotent retries).
	have, err := partialSize(pasteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if offset != have {
		w.Header().Set(uploadproto.HeaderUploadOffset, strconv.FormatInt(have, 10))
		http.Error(w, "offset mismatch", http.StatusConflict)
		return
	}

	// Never let a chunk push the file past the declared total.
	r.Body = http.MaxBytesReader(w, r.Body, total-have)
	f, err := os.OpenFile(partPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf := make([]byte, 64*1024)
	written, copyErr := io.CopyBuffer(f, r.Body, buf)
	closeErr := f.Close()
	if copyErr != nil {
		// Bytes that did land are kept, so the client can resume from the new
		// size; just surface where we are now.
		w.Header().Set(uploadproto.HeaderUploadOffset, strconv.FormatInt(have+written, 10))
		http.Error(w, copyErr.Error(), http.StatusInternalServerError)
		return
	}
	if closeErr != nil {
		http.Error(w, closeErr.Error(), http.StatusInternalServerError)
		return
	}

	newOffset := have + written
	w.Header().Set(uploadproto.HeaderUploadOffset, strconv.FormatInt(newOffset, 10))
	if newOffset < total {
		w.WriteHeader(http.StatusNoContent) // more chunks to come
		return
	}

	// Complete: promote the partial to its final name and record it.
	savedName, err := h.finalize(r, partPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.pasteSvc.UpdatePasteFileName(r.Context(), pasteId, savedName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// finalize atomically renames the completed .part file to its final stored name
// (file_<micros><ext>), deriving the extension from the client-sent filename.
func (h *FileHandler) finalize(r *http.Request, partPath string) (string, error) {
	name := partPath // fallback for ext extraction only
	if raw := r.Header.Get(uploadproto.HeaderUploadFilename); raw != "" {
		if decoded, err := url.QueryUnescape(raw); err == nil {
			name = decoded
		}
	}
	ext := filepath.Ext(filepath.Base(name))
	savedName := fmt.Sprintf("file_%d%s", time.Now().UnixMicro(), ext)
	if err := os.Rename(partPath, parseFilePath(savedName)); err != nil {
		return "", err
	}
	return savedName, nil
}

func partialDirPath() string { return filepath.Join(UploadDir, PartialDir) }

func partialPath(pasteId string) string {
	return filepath.Join(partialDirPath(), filepath.Base(pasteId)+".part")
}

// partialSize returns the bytes already stored for an in-progress upload, or 0
// if none exists yet.
func partialSize(pasteId string) (int64, error) {
	info, err := os.Stat(partialPath(pasteId))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
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
