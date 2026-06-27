package service

import (
	"context"
	"yunyuyuan/anypaste/internal/model"
)

type PasteService struct {
	repo *model.PasteRepo
}

func NewPasteService(repo *model.PasteRepo) *PasteService {
	return &PasteService{repo: repo}
}

func (s *PasteService) CreatePaste(
	ctx context.Context,
	pasteItem *model.Paste,
) (*model.Paste, error) {
	err := s.repo.CreatePaste(ctx, pasteItem)
	if err != nil {
		return nil, err
	}
	return pasteItem, nil
}

func (s *PasteService) GetPaste(
	ctx context.Context,
	id string,
) (*model.Paste, error) {
	paste, err := s.repo.GetPaste(ctx, id)
	if err != nil {
		return nil, err
	}
	return &paste, nil
}

func (s *PasteService) UpdatePasteFileName(
	ctx context.Context,
	id, filename string,
) error {
	_, err := s.repo.UpdatePasteFileName(ctx, id, filename)
	if err != nil {
		return err
	}
	return nil
}

func (s *PasteService) UpdatePaste(
	ctx context.Context,
	id, content string,
) error {
	_, err := s.repo.UpdatePaste(ctx, id, content)
	return err
}

// ReferencedFileNames returns the saved file names still referenced by a paste.
func (s *PasteService) ReferencedFileNames(ctx context.Context) ([]string, error) {
	return s.repo.ReferencedFileNames(ctx)
}

func (s *PasteService) DeletePaste(
	ctx context.Context,
	id string,
) error {
	_, err := s.repo.DeletePaste(ctx, id)
	return err
}

func (s *PasteService) ListPastes(
	ctx context.Context,
) ([]model.Paste, error) {
	return s.repo.ListPastes(ctx)
}
