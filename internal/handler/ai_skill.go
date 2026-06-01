package handler

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AISkillHandler struct {
	svc    *service.AISkillService
	logger *zap.Logger
}

func NewAISkillHandler(svc *service.AISkillService, logger *zap.Logger) *AISkillHandler {
	return &AISkillHandler{svc: svc, logger: logger}
}

type aiSkillRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	Instructions  string            `json:"instructions"`
	License       string            `json:"license"`
	Compatibility string            `json:"compatibility"`
	AllowedTools  string            `json:"allowed_tools"`
	Metadata      map[string]string `json:"metadata"`
	Enabled       *bool             `json:"enabled"`
}

func (h *AISkillHandler) List(c *gin.Context) {
	search := c.Query("search")
	skills, err := h.svc.List(c.Request.Context(), search)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, skills)
}

func (h *AISkillHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	skill, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, skill)
}

func (h *AISkillHandler) Create(c *gin.Context) {
	var req aiSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	userID := GetCurrentUserID(c)

	skill := &model.AISkill{
		Name:          req.Name,
		Description:   req.Description,
		Instructions:  req.Instructions,
		License:       req.License,
		Compatibility: req.Compatibility,
		AllowedTools:  req.AllowedTools,
		Enabled:       true,
		CreatedBy:     strconv.FormatUint(uint64(userID), 10),
		UpdatedBy:     strconv.FormatUint(uint64(userID), 10),
	}
	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}
	if req.Metadata != nil {
		if err := skill.SetMetadataMap(req.Metadata); err != nil {
			Error(c, err)
			return
		}
	}

	if err := h.svc.Create(c.Request.Context(), skill); err != nil {
		Error(c, err)
		return
	}
	Success(c, skill)
}

func (h *AISkillHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	var req aiSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	userID := GetCurrentUserID(c)

	skill := &model.AISkill{
		ID:            uint(id),
		Name:          req.Name,
		Description:   req.Description,
		Instructions:  req.Instructions,
		License:       req.License,
		Compatibility: req.Compatibility,
		AllowedTools:  req.AllowedTools,
		UpdatedBy:     strconv.FormatUint(uint64(userID), 10),
	}
	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}
	if req.Metadata != nil {
		if err := skill.SetMetadataMap(req.Metadata); err != nil {
			Error(c, err)
			return
		}
	}

	if err := h.svc.Update(c.Request.Context(), skill); err != nil {
		Error(c, err)
		return
	}
	Success(c, skill)
}

func (h *AISkillHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// --- File endpoints ---

func (h *AISkillHandler) GetFiles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	files, err := h.svc.GetFiles(c.Request.Context(), uint(id))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, files)
}

func (h *AISkillHandler) GetFile(c *gin.Context) {
	fileID, err := strconv.ParseUint(c.Param("fileId"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	file, err := h.svc.GetFile(c.Request.Context(), uint(fileID))
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, file)
}

type aiSkillFileRequest struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content"`
}

func (h *AISkillHandler) AddFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	var req aiSkillFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}

	file := &model.AISkillFile{
		Name:    req.Name,
		Content: req.Content,
	}
	if err := h.svc.AddFile(c.Request.Context(), uint(id), file); err != nil {
		Error(c, err)
		return
	}
	Success(c, file)
}

func (h *AISkillHandler) DeleteFile(c *gin.Context) {
	fileID, err := strconv.ParseUint(c.Param("fileId"), 10, 64)
	if err != nil {
		Error(c, apperr.ErrInvalidParam)
		return
	}
	if err := h.svc.DeleteFile(c.Request.Context(), uint(fileID)); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Import imports a skill from a .zip or .tar.gz archive.
func (h *AISkillHandler) Import(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	userID := GetCurrentUserID(c)
	userStr := strconv.FormatUint(uint64(userID), 10)

	var skill *model.AISkill
	var files []model.AISkillFile

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".zip" {
		skill, files, err = parseZipArchive(file, header.Size)
	} else if ext == ".gz" && strings.HasSuffix(strings.ToLower(header.Filename), ".tar.gz") {
		skill, files, err = parseTarGzArchive(file)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format, use .zip or .tar.gz"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	skill.CreatedBy = userStr
	skill.UpdatedBy = userStr
	skill.Enabled = true

	if err := h.svc.ImportSkill(c.Request.Context(), skill, files); err != nil {
		Error(c, err)
		return
	}
	Success(c, skill)
}

// parseZipArchive reads a zip file and extracts SKILL.md + auxiliary files.
func parseZipArchive(r io.ReaderAt, size int64) (*model.AISkill, []model.AISkillFile, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, nil, err
	}

	var skillMD string
	var files []model.AISkillFile
	skillDir := ""

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name

		// Detect if SKILL.md is inside a subdirectory
		if strings.HasSuffix(name, "SKILL.md") && skillDir == "" {
			dir := filepath.Dir(name)
			if dir != "." {
				skillDir = dir + "/"
			}
		}

		// Strip the top-level directory prefix for file names
		fileName := name
		if skillDir != "" && strings.HasPrefix(name, skillDir) {
			fileName = strings.TrimPrefix(name, skillDir)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, nil, err
		}

		if filepath.Base(name) == "SKILL.md" {
			skillMD = string(content)
		} else {
			files = append(files, model.AISkillFile{
				Name:    fileName,
				Content: string(content),
			})
		}
	}

	if skillMD == "" {
		return nil, nil, fmt.Errorf("SKILL.md not found in archive")
	}

	skill := parseSKILLMD(skillMD)
	return skill, files, nil
}

// parseTarGzArchive reads a tar.gz file and extracts SKILL.md + auxiliary files.
func parseTarGzArchive(r io.Reader) (*model.AISkill, []model.AISkillFile, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	var skillMD string
	var files []model.AISkillFile
	skillDir := ""

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := hdr.Name

		if strings.HasSuffix(name, "SKILL.md") && skillDir == "" {
			dir := filepath.Dir(name)
			if dir != "." {
				skillDir = dir + "/"
			}
		}

		fileName := name
		if skillDir != "" && strings.HasPrefix(name, skillDir) {
			fileName = strings.TrimPrefix(name, skillDir)
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, nil, err
		}

		if filepath.Base(name) == "SKILL.md" {
			skillMD = string(content)
		} else {
			files = append(files, model.AISkillFile{
				Name:    fileName,
				Content: string(content),
			})
		}
	}

	if skillMD == "" {
		return nil, nil, fmt.Errorf("SKILL.md not found in archive")
	}

	skill := parseSKILLMD(skillMD)
	return skill, files, nil
}

// parseSKILLMD extracts frontmatter fields from a SKILL.md file.
func parseSKILLMD(content string) *model.AISkill {
	skill := &model.AISkill{
		Enabled: true,
	}

	// Parse YAML frontmatter (between --- markers)
	if !strings.HasPrefix(content, "---") {
		skill.Instructions = content
		return skill
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		skill.Instructions = content
		return skill
	}

	frontmatter := parts[1]
	skill.Instructions = strings.TrimSpace(parts[2])

	// Simple key-value parser for frontmatter
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		// Remove surrounding quotes
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}

		switch key {
		case "name":
			skill.Name = val
		case "description":
			skill.Description = val
		case "license":
			skill.License = val
		case "compatibility":
			skill.Compatibility = val
		case "allowed-tools":
			skill.AllowedTools = val
		case "max_iterations":
			// ignore, not stored in our model
		}
	}

	if skill.Name == "" {
		skill.Name = "imported-skill"
	}
	return skill
}
