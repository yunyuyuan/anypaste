package model

import (
	"context"
	"time"
	"yunyuyuan/anypaste/internal/generated"
	"yunyuyuan/anypaste/internal/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Paste represents a single paste entry.
type Paste struct {
	// ID is the primary key, a UUID generated on creation.
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	// Content holds the text content (text pastes) or the stored file path (file pastes).
	Content string `gorm:"type:text;not null" json:"content"`
	// FileName is the original uploaded file name; set only for file pastes.
	FileName *string `gorm:"type:text" json:"file_name,omitempty"`
	// ViewPasswd is an optional password required to view the paste.
	ViewPasswd *string `gorm:"type:text" json:"view_passwd,omitempty"`
	// ExpiredAt is an optional expiration time. Nil means it never expires.
	ExpiredAt *time.Time `json:"expired_at,omitempty"`

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

// notExpired matches pastes that are still valid: either they never expire
// (expired_at IS NULL) or their expiration is still in the future. Defined once
// here so every read query shares the exact same semantics.
func notExpired() clause.Expression {
	return clause.Or(
		generated.Paste.ExpiredAt.IsNull(),
		generated.Paste.ExpiredAt.Gt(time.Now()),
	)
}

func (r *PasteRepo) CreatePaste(ctx context.Context, p *Paste) error {
	return gorm.G[Paste](r.db).Create(ctx, p)
}

func (r *PasteRepo) GetPaste(ctx context.Context, id string) (Paste, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id), notExpired()).First(ctx)
}

func (r *PasteRepo) DeletePaste(ctx context.Context, id string) (int, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).Delete(ctx)
}

func (r *PasteRepo) UpdatePasteFileName(ctx context.Context, id, filename string) (int, error) {
	return gorm.G[Paste](r.db).Where(generated.Paste.ID.Eq(id)).Update(ctx, generated.Paste.FileName.Column().Name, filename)
}

// UpdatePaste updates a paste's content and expiration in one statement.
// A nil expiredAt clears the expiration (the paste then never expires); only
// non-expired pastes can be updated, matching the read-side visibility rule.
func (r *PasteRepo) UpdatePaste(ctx context.Context, id, content string, expiredAt *time.Time) (int, error) {
	// nil Value renders as "expired_at = NULL", which the typed Set(time.Time) cannot express.
	expiredAtAssign := clause.Assignment{Column: generated.Paste.ExpiredAt.Column()}
	if expiredAt != nil {
		expiredAtAssign = generated.Paste.ExpiredAt.Set(*expiredAt)
	}
	return gorm.G[Paste](r.db).
		Where(generated.Paste.ID.Eq(id), notExpired()).
		Set(
			generated.Paste.Content.Set(content),
			expiredAtAssign,
		).
		Update(ctx)
}

func (r *PasteRepo) ListPastes(ctx context.Context) ([]Paste, error) {
	return gorm.G[Paste](r.db).Where(notExpired()).Find(ctx)
}
