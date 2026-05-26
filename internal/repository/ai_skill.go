package repository

import (
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

func (r *AISkillRepository) Create(skill *model.AISkill) error {
	return r.db.Create(skill).Error
}

func (r *AISkillRepository) GetByID(id uint) (*model.AISkill, error) {
	var skill model.AISkill
	err := r.db.First(&skill, id).Error
	return &skill, err
}

func (r *AISkillRepository) Update(skill *model.AISkill) error {
	return r.db.Save(skill).Error
}

func (r *AISkillRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_id = ?", id).Delete(&model.AISkillFile{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AISkill{}, id).Error
	})
}

func (r *AISkillRepository) List(search string) ([]model.AISkill, error) {
	var skills []model.AISkill
	q := r.db.Order("id DESC")
	if search != "" {
		q = q.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	err := q.Find(&skills).Error
	return skills, err
}

// --- File operations ---

func (r *AISkillRepository) CreateFile(file *model.AISkillFile) error {
	file.Size = int64(len(file.Content))
	return r.db.Create(file).Error
}

func (r *AISkillRepository) GetFileByID(id uint) (*model.AISkillFile, error) {
	var file model.AISkillFile
	err := r.db.First(&file, id).Error
	return &file, err
}

func (r *AISkillRepository) GetFilesBySkillID(skillID uint) ([]model.AISkillFile, error) {
	var files []model.AISkillFile
	err := r.db.Where("skill_id = ?", skillID).Order("name").Find(&files).Error
	return files, err
}

func (r *AISkillRepository) DeleteFile(id uint) error {
	return r.db.Delete(&model.AISkillFile{}, id).Error
}

func (r *AISkillRepository) DeleteFilesBySkillID(skillID uint) error {
	return r.db.Where("skill_id = ?", skillID).Delete(&model.AISkillFile{}).Error
}

// BatchUpsertFiles replaces all files for a skill (delete stale, upsert new).
func (r *AISkillRepository) BatchUpsertFiles(skillID uint, files []model.AISkillFile) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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
