package repository

import (
	"context"

	"github.com/sreagent/sreagent/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AISkillRepository struct {
	db *gorm.DB
}

func NewAISkillRepository(db *gorm.DB) *AISkillRepository {
	return &AISkillRepository{db: db}
}

func (r *AISkillRepository) Create(ctx context.Context, skill *model.AISkill) error {
	return r.db.WithContext(ctx).Create(skill).Error
}

func (r *AISkillRepository) GetByID(ctx context.Context, id uint) (*model.AISkill, error) {
	var skill model.AISkill
	err := r.db.WithContext(ctx).First(&skill, id).Error
	return &skill, err
}

func (r *AISkillRepository) Update(ctx context.Context, skill *model.AISkill) error {
	return r.db.WithContext(ctx).Save(skill).Error
}

func (r *AISkillRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_id = ?", id).Delete(&model.AISkillFile{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AISkill{}, id).Error
	})
}

func (r *AISkillRepository) List(ctx context.Context, search string) ([]model.AISkill, error) {
	var skills []model.AISkill
	q := r.db.WithContext(ctx).Order("id DESC")
	if search != "" {
		q = q.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	err := q.Find(&skills).Error
	return skills, err
}

// --- File operations ---

func (r *AISkillRepository) CreateFile(ctx context.Context, file *model.AISkillFile) error {
	file.Size = int64(len(file.Content))
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *AISkillRepository) GetFileByID(ctx context.Context, id uint) (*model.AISkillFile, error) {
	var file model.AISkillFile
	err := r.db.WithContext(ctx).First(&file, id).Error
	return &file, err
}

func (r *AISkillRepository) GetFilesBySkillID(ctx context.Context, skillID uint) ([]model.AISkillFile, error) {
	var files []model.AISkillFile
	err := r.db.WithContext(ctx).Where("skill_id = ?", skillID).Order("name").Find(&files).Error
	return files, err
}

func (r *AISkillRepository) DeleteFile(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AISkillFile{}, id).Error
}

func (r *AISkillRepository) DeleteFilesBySkillID(ctx context.Context, skillID uint) error {
	return r.db.WithContext(ctx).Where("skill_id = ?", skillID).Delete(&model.AISkillFile{}).Error
}

// BatchUpsertFiles replaces all files for a skill (delete stale, upsert new).
func (r *AISkillRepository) BatchUpsertFiles(ctx context.Context, skillID uint, files []model.AISkillFile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete files not in the new list
		if len(files) > 0 {
			ids := make([]uint, 0, len(files))
			for _, f := range files {
				if f.ID > 0 {
					ids = append(ids, f.ID)
				}
			}
			if len(ids) > 0 {
				if err := tx.Where("skill_id = ? AND id NOT IN ?", skillID, ids).Delete(&model.AISkillFile{}).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Where("skill_id = ?", skillID).Delete(&model.AISkillFile{}).Error; err != nil {
					return err
				}
			}
		} else {
			if err := tx.Where("skill_id = ?", skillID).Delete(&model.AISkillFile{}).Error; err != nil {
				return err
			}
		}

		// Upsert each file
		for i := range files {
			files[i].SkillID = skillID
			files[i].Size = int64(len(files[i].Content))
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "content", "size", "updated_at"}),
			}).Create(&files[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
