package handler

import (
	pastev1 "yunyuyuan/anypaste/gen/paste/v1"
	"yunyuyuan/anypaste/internal/model"
)

func PasteModelToProto(pasteItem *model.Paste) *pastev1.PasteItem {
	return &pastev1.PasteItem{
		Id:       pasteItem.ID,
		Content:  pasteItem.Content,
		FileName: pasteItem.FileName,
	}
}
