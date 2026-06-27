package handler

import (
	pastev1 "yunyuyuan/anypaste/gen/paste/v1"
	"yunyuyuan/anypaste/internal/model"
)

func PasteModelToProto(pasteItem *model.Paste) *pastev1.PasteItem {
	var expiredAt *int64
	if pasteItem.ExpiredAt != nil {
		v := pasteItem.ExpiredAt.UnixMilli()
		expiredAt = &v
	}
	return &pastev1.PasteItem{
		Id:         pasteItem.ID,
		Content:    pasteItem.Content,
		ViewPasswd: pasteItem.ViewPasswd,
		ExpiredAt:  expiredAt,
		FileName:   pasteItem.FileName,
	}
}
