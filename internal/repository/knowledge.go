package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type KnowledgeRepository struct {
	db *gorm.DB
}

func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

func (r *KnowledgeRepository) Create(ctx context.Context, doc *model.KnowledgeDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *KnowledgeRepository) GetByID(ctx context.Context, id uint) (*model.KnowledgeDocument, error) {
	var doc model.KnowledgeDocument
	err := r.db.WithContext(ctx).First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *KnowledgeRepository) Update(ctx context.Context, doc *model.KnowledgeDocument) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

func (r *KnowledgeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.KnowledgeDocument{}, id).Error
}

// Search performs a FULLTEXT search on title + content + summary.
func (r *KnowledgeRepository) Search(ctx context.Context, query string, source string, topK int) ([]*model.KnowledgeDocument, error) {
	q := r.db.WithContext(ctx).
		Where("status = ?", "active")

	if source != "" {
		q = q.Where("source = ?", source)
	}

	if query != "" {
		q = q.Where("MATCH(title, content, summary) AGAINST(? IN BOOLEAN MODE)", query)
	}

	if topK <= 0 {
		topK = 10
	}

	var docs []*model.KnowledgeDocument
	err := q.Order("helpful_count DESC, view_count DESC").
		Limit(topK).
		Find(&docs).Error
	return docs, err
}

// List returns paginated knowledge documents.
func (r *KnowledgeRepository) List(ctx context.Context, source string, page, pageSize int) ([]model.KnowledgeDocument, int64, error) {
	var list []model.KnowledgeDocument
	var total int64

	q := r.db.WithContext(ctx).Where("status = ?", "active")
	if source != "" {
		q = q.Where("source = ?", source)
	}

	if err := q.Model(&model.KnowledgeDocument{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// IncrementViewCount atomically increments view_count.
func (r *KnowledgeRepository) IncrementViewCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.KnowledgeDocument{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementHelpfulCount atomically increments helpful_count.
func (r *KnowledgeRepository) IncrementHelpfulCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.KnowledgeDocument{}).
		Where("id = ?", id).
		UpdateColumn("helpful_count", gorm.Expr("helpful_count + 1")).Error
}
