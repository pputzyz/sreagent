package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// Engine executes event pipelines.
type Engine struct {
	pipelineRepo *repository.EventPipelineRepository
	execRepo     *repository.EventPipelineExecutionRepository
	logger       *zap.Logger
}

// NewEngine creates a new pipeline Engine.
func NewEngine(
	pipelineRepo *repository.EventPipelineRepository,
	execRepo *repository.EventPipelineExecutionRepository,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		pipelineRepo: pipelineRepo,
		execRepo:     execRepo,
		logger:       logger,
	}
}

// Execute runs a pipeline against an alert event.
// Returns the processed event (nil if dropped), the execution record, and any error.
func (e *Engine) Execute(ctx context.Context, pipelineObj *model.EventPipeline, event *model.AlertEvent, triggerBy string) (*model.AlertEvent, *model.EventPipelineExecution, error) {
	startTime := time.Now()

	exec := &model.EventPipelineExecution{
		ID:           uuid.New().String(),
		PipelineID:   pipelineObj.ID,
		PipelineName: pipelineObj.Name,
		Mode:         "event",
		Status:       "running",
		TriggerBy:    triggerBy,
		CreatedAt:    startTime,
	}
	if event != nil {
		exec.EventID = event.ID
	}

	// Create execution record
	if err := e.execRepo.Create(ctx, exec); err != nil {
		e.logger.Error("failed to create pipeline execution record",
			zap.Uint("pipeline_id", pipelineObj.ID),
			zap.Error(err),
		)
		// Continue execution even if we can't record it
	}

	var nodeResults []model.NodeResult
	currentEvent := event
	dropped := false

	for _, pc := range pipelineObj.ProcessorConfigs {
		nodeStart := time.Now()
		proc, err := Get(pc.Typ, pc.Config)
		if err != nil {
			result := model.NodeResult{
				ProcessorType: pc.Typ,
				Status:        "failed",
				Message:       fmt.Sprintf("failed to create processor: %v", err),
				DurationMs:    time.Since(nodeStart).Milliseconds(),
			}
			nodeResults = append(nodeResults, result)

			exec.Status = "failed"
			exec.ErrorMessage = result.Message
			e.finishExecution(ctx, exec, nodeResults, startTime)
			return currentEvent, exec, fmt.Errorf("pipeline processor %q init failed: %w", pc.Typ, err)
		}

		processedEvent, msg, procErr := proc.Process(ctx, currentEvent)
		result := model.NodeResult{
			ProcessorType: pc.Typ,
			Message:       msg,
			DurationMs:    time.Since(nodeStart).Milliseconds(),
		}

		if procErr != nil {
			result.Status = "failed"
			nodeResults = append(nodeResults, result)

			exec.Status = "failed"
			exec.ErrorMessage = procErr.Error()
			e.finishExecution(ctx, exec, nodeResults, startTime)
			return currentEvent, exec, procErr
		}

		if processedEvent == nil {
			// Event was dropped
			result.Status = "dropped"
			nodeResults = append(nodeResults, result)
			dropped = true
			currentEvent = nil
			break
		}

		result.Status = "success"
		nodeResults = append(nodeResults, result)
		currentEvent = processedEvent
	}

	if dropped {
		exec.Status = "terminated"
		exec.ErrorMessage = "event dropped by processor"
	} else {
		exec.Status = "success"
	}
	e.finishExecution(ctx, exec, nodeResults, startTime)

	return currentEvent, exec, nil
}

func (e *Engine) finishExecution(ctx context.Context, exec *model.EventPipelineExecution, nodeResults []model.NodeResult, startTime time.Time) {
	exec.FinishedAt = time.Now()
	exec.DurationMs = time.Since(startTime).Milliseconds()

	if nodeResults != nil {
		data, _ := json.Marshal(nodeResults)
		exec.NodeResults = string(data)
	}

	if err := e.execRepo.Update(ctx, exec); err != nil {
		e.logger.Error("failed to update pipeline execution record",
			zap.String("exec_id", exec.ID),
			zap.Error(err),
		)
	}
}
