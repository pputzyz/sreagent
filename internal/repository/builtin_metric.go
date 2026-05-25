package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type BuiltinMetricRepository struct {
	db *gorm.DB
}

func NewBuiltinMetricRepository(db *gorm.DB) *BuiltinMetricRepository {
	return &BuiltinMetricRepository{db: db}
}

func (r *BuiltinMetricRepository) Create(ctx context.Context, m *model.BuiltinMetric) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *BuiltinMetricRepository) GetByID(ctx context.Context, id uint) (*model.BuiltinMetric, error) {
	var m model.BuiltinMetric
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	m.DB2FE()
	return &m, nil
}

func (r *BuiltinMetricRepository) List(ctx context.Context, collector, typ, query, unit, lang string, page, pageSize int) ([]model.BuiltinMetric, int64, error) {
	var metrics []model.BuiltinMetric
	var total int64

	q := r.db.WithContext(ctx).Model(&model.BuiltinMetric{})
	if collector != "" {
		q = q.Where("collector = ?", collector)
	}
	if typ != "" {
		q = q.Where("typ = ?", typ)
	}
	if unit != "" {
		q = q.Where("unit = ?", unit)
	}
	if lang != "" {
		q = q.Where("lang = ?", lang)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(name LIKE ? OR note LIKE ? OR expression LIKE ?)", like, like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("collector, typ, name").Offset(offset).Limit(pageSize).Find(&metrics).Error; err != nil {
		return nil, 0, err
	}

	for i := range metrics {
		metrics[i].DB2FE()
	}
	return metrics, total, nil
}

func (r *BuiltinMetricRepository) Update(ctx context.Context, m *model.BuiltinMetric) error {
	return r.db.WithContext(ctx).Model(m).Select("*").Updates(m).Error
}

func (r *BuiltinMetricRepository) Delete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.BuiltinMetric{}).Error
}

func (r *BuiltinMetricRepository) GetTypes(ctx context.Context, collector, query, lang string) ([]string, error) {
	var types []string
	q := r.db.WithContext(ctx).Model(&model.BuiltinMetric{}).Distinct("typ")
	if collector != "" {
		q = q.Where("collector = ?", collector)
	}
	if lang != "" {
		q = q.Where("lang = ?", lang)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(name LIKE ? OR note LIKE ?)", like, like)
	}
	if err := q.Pluck("typ", &types).Error; err != nil {
		return nil, err
	}
	return types, nil
}

func (r *BuiltinMetricRepository) GetCollectors(ctx context.Context, typ, query, lang string) ([]string, error) {
	var collectors []string
	q := r.db.WithContext(ctx).Model(&model.BuiltinMetric{}).Distinct("collector")
	if typ != "" {
		q = q.Where("typ = ?", typ)
	}
	if lang != "" {
		q = q.Where("lang = ?", lang)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(name LIKE ? OR note LIKE ?)", like, like)
	}
	if err := q.Pluck("collector", &collectors).Error; err != nil {
		return nil, err
	}
	return collectors, nil
}

// MetricFilter repository

type MetricFilterRepository struct {
	db *gorm.DB
}

func NewMetricFilterRepository(db *gorm.DB) *MetricFilterRepository {
	return &MetricFilterRepository{db: db}
}

func (r *MetricFilterRepository) Create(ctx context.Context, f *model.MetricFilter) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *MetricFilterRepository) List(ctx context.Context, createdBy string) ([]model.MetricFilter, error) {
	var filters []model.MetricFilter
	if err := r.db.WithContext(ctx).Where("created_by = ?", createdBy).Order("name").Find(&filters).Error; err != nil {
		return nil, err
	}
	for i := range filters {
		filters[i].DB2FE()
	}
	return filters, nil
}

func (r *MetricFilterRepository) Update(ctx context.Context, f *model.MetricFilter) error {
	return r.db.WithContext(ctx).Model(f).Select("*").Updates(f).Error
}

func (r *MetricFilterRepository) Delete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.MetricFilter{}).Error
}
