package handler

import (
	"context"
	"os"
	pastev1 "yunyuyuan/anypaste/gen/paste/v1"
	"yunyuyuan/anypaste/internal/model"
	"yunyuyuan/anypaste/internal/service"
	"yunyuyuan/anypaste/internal/utils"
)

type PasteHandler struct {
	svc *service.PasteService
}

func NewPasteHandler(svc *service.PasteService) *PasteHandler {
	return &PasteHandler{svc: svc}
}

func (h *PasteHandler) CreatePaste(
	ctx context.Context,
	req *pastev1.CreatePasteRequest,
) (*pastev1.CreatePasteResponse, error) {
	pasteItem, err := h.svc.CreatePaste(ctx, &model.Paste{
		Content:    req.Content,
		ViewPasswd: req.ViewPasswd,
		ExpiredAt:  utils.TimestampToTime(req.ExpiredAt),
	})
	if err != nil {
		return nil, err
	}
	return &pastev1.CreatePasteResponse{
		Success: true,
		Id:      pasteItem.ID,
	}, nil
}

func (h *PasteHandler) UpdatePaste(
	ctx context.Context,
	req *pastev1.UpdatePasteRequest,
) (*pastev1.UpdatePasteResponse, error) {
	if err := h.svc.UpdatePaste(ctx, req.Id, req.Content, utils.TimestampToTime(req.ExpiredAt)); err != nil {
		return nil, err
	}
	return &pastev1.UpdatePasteResponse{
		Success: true,
	}, nil
}

func (h *PasteHandler) DeletePaste(
	ctx context.Context,
	req *pastev1.DeletePasteRequest,
) (*pastev1.DeletePasteResponse, error) {
	pasteItem, err := h.svc.GetPaste(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	// 文件类型同时删除磁盘上的文件，best-effort：删除失败不阻塞记录删除
	if pasteItem.FileName != nil && *pasteItem.FileName != "" {
		_ = os.Remove(parseFilePath(*pasteItem.FileName))
	}
	if err := h.svc.DeletePaste(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pastev1.DeletePasteResponse{
		Success: true,
	}, nil
}

func (h *PasteHandler) ListPastes(
	ctx context.Context,
	req *pastev1.ListPastesRequest,
) (*pastev1.ListPastesResponse, error) {
	result, err := h.svc.ListPastes(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*pastev1.PasteItem, len(result))
	for i := range result {
		list[i] = PasteModelToProto(&result[i])
	}
	return &pastev1.ListPastesResponse{
		List: list,
	}, nil
}
