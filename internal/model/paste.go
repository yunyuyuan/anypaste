package model

import (
	"context"
	"time"
	"yunyuyuan/anypaste/internal/generated"
	"yunyuyuan/anypaste/internal/utils"

	"gorm.io/gorm"
)

// Paste represents a single paste entry.
type Paste struct {
	// ID is the primary key, a UUID generated on creation.
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	// Content holds the text content (text pastes) or the stored file path (file pastes).
	Content string `gorm:"type:text;not null" json:"content"`
	// FileName is the original uploaded file name; set only for file pastes.
	FileName *string `gorm:"type:text" json:"file_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate generates a UUID for the primary key if one is not set.
func (p *Paste) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = utils.RandomStr(6)
	}
	return nil
}

type PasteRepo struct {
	db *gorm.DB
}

func NewPasteRepo(db *gorm.DB) *PasteRepo {
	return &PasteRepo{db: db}
}

func (r *PasteRepo) CreatePaste(ctx context.Context, p *Paste) error {
	return gorm.G[Paste](r.db).Create(ctx, p)
}

func (r *PasteRepo) GetPaste(ctx context.Context, id string) (Paste, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).First(ctx)
}

func (r *PasteRepo) DeletePaste(ctx context.Context, id string) (int, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).Delete(ctx)
}

func (r *PasteRepo) UpdatePasteFileName(ctx context.Context, id, filename string) (int, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).Update(ctx, generated.Paste.FileName.Column().Name, filename)
}

func (r *PasteRepo) UpdatePaste(ctx context.Context, id, content string) (int, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).Update(ctx, generated.Paste.Content.Column().Name, content)
}

func (r *PasteRepo) ListPastes(ctx context.Context) ([]Paste, error) {
	return gorm.G[Paste](r.db).Find(ctx)
}

// ReferencedFileNames returns every saved file name still referenced by a paste,
// used by the cleanup job to find orphaned files in the uploads dir.
func (r *PasteRepo) ReferencedFileNames(ctx context.Context) ([]string, error) {
	pastes, err := gorm.G[Paste](r.db).Find(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(pastes))
	for i := range pastes {
		if pastes[i].FileName != nil && *pastes[i].FileName != "" {
			names = append(names, *pastes[i].FileName)
		}
	}
	return names, nil
}
