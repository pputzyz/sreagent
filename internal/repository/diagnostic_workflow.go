package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type DiagnosticWorkflowRepository struct {
	db *gorm.DB
}

func NewDiagnosticWorkflowRepository(db *gorm.DB) *DiagnosticWorkflowRepository {
	return &DiagnosticWorkflowRepository{db: db}
}

// --- Workflow CRUD ---

func (r *DiagnosticWorkflowRepository) Create(ctx context.Context, wf *model.DiagnosticWorkflow) error {
	return r.db.WithContext(ctx).Create(wf).Error
}

func (r *DiagnosticWorkflowRepository) GetByID(ctx context.Context, id uint) (*model.DiagnosticWorkflow, error) {
	var wf model.DiagnosticWorkflow
	err := r.db.WithContext(ctx).First(&wf, id).Error
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *DiagnosticWorkflowRepository) List(ctx context.Context, category string, enabled *bool, page, pageSize int) ([]model.DiagnosticWorkflow, int64, error) {
	var list []model.DiagnosticWorkflow
	var total int64

	q := r.db.WithContext(ctx)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}

	if err := q.Model(&model.DiagnosticWorkflow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *DiagnosticWorkflowRepository) Update(ctx context.Context, wf *model.DiagnosticWorkflow) error {
	return r.db.WithContext(ctx).Save(wf).Error
}

func (r *DiagnosticWorkflowRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DiagnosticWorkflow{}, id).Error
}

// --- Steps CRUD ---

func (r *DiagnosticWorkflowRepository) CreateStep(ctx context.Context, step *model.DiagnosticWorkflowStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DiagnosticWorkflowRepository) ListSteps(ctx context.Context, workflowID uint) ([]model.DiagnosticWorkflowStep, error) {
	var steps []model.DiagnosticWorkflowStep
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("step_order ASC").
		Find(&steps).Error
	return steps, err
}

func (r *DiagnosticWorkflowRepository) DeleteSteps(ctx context.Context, workflowID uint) error {
	return r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Delete(&model.DiagnosticWorkflowStep{}).Error
}

// ReplaceSteps atomically replaces all steps for a workflow within a single transaction.
func (r *DiagnosticWorkflowRepository) ReplaceSteps(ctx context.Context, workflowID uint, steps []model.DiagnosticWorkflowStep) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workflow_id = ?", workflowID).Delete(&model.DiagnosticWorkflowStep{}).Error; err != nil {
			return err
		}
		for i := range steps {
			steps[i].WorkflowID = workflowID
			steps[i].StepOrder = i + 1
			if err := tx.Create(&steps[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Run CRUD ---

func (r *DiagnosticWorkflowRepository) CreateRun(ctx context.Context, run *model.DiagnosticRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *DiagnosticWorkflowRepository) GetRun(ctx context.Context, id uint) (*model.DiagnosticRun, error) {
	var run model.DiagnosticRun
	err := r.db.WithContext(ctx).First(&run, id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *DiagnosticWorkflowRepository) UpdateRun(ctx context.Context, run *model.DiagnosticRun) error {
	return r.db.WithContext(ctx).Save(run).Error
}

func (r *DiagnosticWorkflowRepository) ListRuns(ctx context.Context, workflowID *uint, incidentID *uint, status string, page, pageSize int) ([]model.DiagnosticRun, int64, error) {
	var list []model.DiagnosticRun
	var total int64

	q := r.db.WithContext(ctx)
	if workflowID != nil {
		q = q.Where("workflow_id = ?", *workflowID)
	}
	if incidentID != nil {
		q = q.Where("incident_id = ?", *incidentID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Model(&model.DiagnosticRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// --- Run Step CRUD ---

func (r *DiagnosticWorkflowRepository) CreateRunStep(ctx context.Context, step *model.DiagnosticRunStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DiagnosticWorkflowRepository) UpdateRunStep(ctx context.Context, step *model.DiagnosticRunStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

func (r *DiagnosticWorkflowRepository) ListRunSteps(ctx context.Context, runID uint) ([]model.DiagnosticRunStep, error) {
	var steps []model.DiagnosticRunStep
	err := r.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("step_order ASC").
		Find(&steps).Error
	return steps, err
}

// FindMatchingWorkflows finds enabled workflows whose trigger_labels match the given labels.
func (r *DiagnosticWorkflowRepository) FindMatchingWorkflows(ctx context.Context, labels map[string]string, severity string) ([]model.DiagnosticWorkflow, error) {
	var workflows []model.DiagnosticWorkflow
	q := r.db.WithContext(ctx).Where("enabled = ?", true)

	if severity != "" {
		q = q.Where("trigger_severity = ? OR trigger_severity IS NULL", severity)
	}

	if err := q.Find(&workflows).Error; err != nil {
		return nil, err
	}

	// Filter by label match in Go (JSON_CONTAINS is MySQL-specific)
	var matched []model.DiagnosticWorkflow
	for _, wf := range workflows {
		if wf.TriggerLabels == nil || len(wf.TriggerLabels) == 0 {
			matched = append(matched, wf) // no label filter = matches all
			continue
		}
		allMatch := true
		for k, v := range wf.TriggerLabels {
			if lv, ok := labels[k]; !ok || lv != v {
				allMatch = false
				break
			}
		}
		if allMatch {
			matched = append(matched, wf)
		}
	}

	return matched, nil
}
