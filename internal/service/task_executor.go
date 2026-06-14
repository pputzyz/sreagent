package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
)

// maxConcurrentSSH caps simultaneous SSH connections within a single task batch,
// preventing fd exhaustion / network overload on large "all at once" batches.
const maxConcurrentSSH = 50

// TaskExecutor executes task templates against remote hosts via SSH.
type TaskExecutor struct {
	tplRepo        *repository.TaskTplRepository
	recRepo        *repository.TaskRecordRepository
	logger         *zap.Logger
	knownHostsFile string // path to SSH known_hosts file
}

// NewTaskExecutor creates a new TaskExecutor.
func NewTaskExecutor(
	tplRepo *repository.TaskTplRepository,
	recRepo *repository.TaskRecordRepository,
	logger *zap.Logger,
	knownHostsFile string,
) *TaskExecutor {
	if knownHostsFile == "" {
		knownHostsFile = "/etc/ssh/ssh_known_hosts"
	}
	return &TaskExecutor{
		tplRepo:        tplRepo,
		recRepo:        recRepo,
		logger:         logger,
		knownHostsFile: knownHostsFile,
	}
}

// ExecuteTaskRequest contains the parameters for executing a task.
type ExecuteTaskRequest struct {
	TplID   uint     `json:"tpl_id"`
	Hosts   []string `json:"hosts"`    // override hosts (optional, uses tpl hosts if empty)
	EventID uint     `json:"event_id"` // 0 if manual
	Title   string   `json:"title"`    // override title (optional)
}

// ExecuteTask creates a TaskRecord and spawns goroutines to execute on each host.
func (e *TaskExecutor) ExecuteTask(ctx context.Context, req *ExecuteTaskRequest, userID uint) (*model.TaskRecord, error) {
	// Load template
	tpl, err := e.tplRepo.GetByID(ctx, req.TplID)
	if err != nil {
		return nil, fmt.Errorf("task template not found: %w", err)
	}

	// Determine hosts
	hosts := req.Hosts
	if len(hosts) == 0 && tpl.Hosts != "" {
		if err := json.Unmarshal([]byte(tpl.Hosts), &hosts); err != nil {
			return nil, fmt.Errorf("failed to parse template hosts: %w", err)
		}
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts specified")
	}

	// Determine title
	title := req.Title
	if title == "" {
		title = tpl.Name
	}

	// Marshal hosts to JSON for storage
	hostsJSON, _ := json.Marshal(hosts)

	// Create task record
	record := &model.TaskRecord{
		TplID:     tpl.ID,
		EventID:   req.EventID,
		Title:     title,
		Account:   tpl.Account,
		Password:  tpl.Password,
		Batch:     tpl.Batch,
		Tolerance: tpl.Tolerance,
		Timeout:   tpl.Timeout,
		Script:    tpl.Script,
		Args:      tpl.Args,
		Hosts:     string(hostsJSON),
		Status:    model.TaskStatusRunning,
		CreateBy:  strconv.FormatUint(uint64(userID), 10),
	}

	if err := e.recRepo.CreateRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}

	// Execute asynchronously
	go e.execute(record, hosts, tpl)

	return record, nil
}

// ExecuteDirect executes a task directly from a form (without a template).
func (e *TaskExecutor) ExecuteDirect(ctx context.Context, script, args, account string, timeout, batch, tolerance int, hosts []string, title string, userID uint, eventID uint) (*model.TaskRecord, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts specified")
	}
	if timeout <= 0 {
		timeout = 60
	}

	hostsJSON, _ := json.Marshal(hosts)

	record := &model.TaskRecord{
		EventID:   eventID,
		Title:     title,
		Account:   account,
		Batch:     batch,
		Tolerance: tolerance,
		Timeout:   timeout,
		Script:    script,
		Args:      args,
		Hosts:     string(hostsJSON),
		Status:    model.TaskStatusRunning,
		CreateBy:  strconv.FormatUint(uint64(userID), 10),
	}

	if err := e.recRepo.CreateRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}

	go e.execute(record, hosts, nil)

	return record, nil
}

// execute runs the task on all hosts, respecting batch mode.
func (e *TaskExecutor) execute(record *model.TaskRecord, hosts []string, tpl *model.TaskTpl) {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("task executor panic",
				zap.Uint("task_id", record.ID),
				zap.Any("panic", r))
			// Mark task as failed
			record.Status = model.TaskStatusFail
			_ = e.recRepo.UpdateRecord(context.Background(), record)
		}
	}()
	ctx := context.Background()

	// Create host records
	hostRecords := make([]*model.TaskHostRecord, len(hosts))
	for i, host := range hosts {
		hr := &model.TaskHostRecord{
			TaskID: record.ID,
			Host:   host,
			Status: model.TaskStatusPending,
		}
		if err := e.recRepo.CreateHostRecord(ctx, hr); err != nil {
			e.logger.Error("failed to create host record",
				zap.Uint("task_id", record.ID),
				zap.String("host", host),
				zap.Error(err),
			)
		}
		hostRecords[i] = hr
	}

	batch := record.Batch
	if batch <= 0 {
		batch = len(hosts) // all at once
	}

	var pauseDuration time.Duration
	pauseStr := ""
	if tpl != nil {
		pauseStr = tpl.Pause
	}
	if pauseStr != "" {
		pauseStr = strings.ReplaceAll(pauseStr, "，", ",")
		parts := strings.Split(pauseStr, ",")
		if len(parts) >= 2 {
			// Format: "batch_size,pause_seconds"
			if p, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				pauseDuration = time.Duration(p) * time.Second
			}
		}
	}

	// Execute in batches. Cap concurrent SSH connections per batch with a semaphore so
	// a large "all at once" batch can't exhaust file descriptors / overwhelm the network.
	var totalFailed int
	sem := make(chan struct{}, maxConcurrentSSH)
	for i := 0; i < len(hosts); i += batch {
		end := i + batch
		if end > len(hosts) {
			end = len(hosts)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for j := i; j < end; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Per-host recover: a panic in one host's execution must not crash the process.
				defer func() {
					if r := recover(); r != nil {
						e.logger.Error("task host execution panic recovered",
							zap.String("host", hostRecords[idx].Host), zap.Any("recover", r))
						hostRecords[idx].Status = model.TaskStatusFail
						mu.Lock()
						totalFailed++
						mu.Unlock()
					}
				}()
				sem <- struct{}{}
				defer func() { <-sem }()
				e.executeOnHost(ctx, record, hostRecords[idx])
				mu.Lock()
				if hostRecords[idx].Status == model.TaskStatusFail {
					totalFailed++
				}
				mu.Unlock()
			}(j)
		}

		wg.Wait()

		// Pause between batches
		if pauseDuration > 0 && end < len(hosts) {
			time.Sleep(pauseDuration)
		}
	}

	// Determine aggregate status
	finalStatus := model.TaskStatusSuccess
	if totalFailed > record.Tolerance {
		finalStatus = model.TaskStatusFail
	}

	record.Status = finalStatus
	if err := e.recRepo.UpdateRecord(ctx, record); err != nil {
		e.logger.Error("failed to update task record status",
			zap.Uint("task_id", record.ID),
			zap.Error(err),
		)
	}

	e.logger.Info("task execution completed",
		zap.Uint("task_id", record.ID),
		zap.String("title", record.Title),
		zap.Int("hosts", len(hosts)),
		zap.Int("failed", totalFailed),
		zap.Int("status", finalStatus),
	)
}

// executeOnHost runs the script on a single host via SSH.
func (e *TaskExecutor) executeOnHost(ctx context.Context, record *model.TaskRecord, hr *model.TaskHostRecord) {
	hr.Status = model.TaskStatusRunning
	_ = e.recRepo.UpdateHostRecord(ctx, hr)

	start := time.Now()

	timeout := record.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	password := record.Password
	if password == "" {
		hr.Status = model.TaskStatusFail
		hr.Stderr = "SSH password not configured for this task — refusing to connect with empty credentials"
		e.logger.Error("SSH password not set, aborting execution",
			zap.Uint("task_id", record.ID),
			zap.String("host", hr.Host),
			zap.String("account", record.Account),
		)
		if err := e.recRepo.UpdateHostRecord(ctx, hr); err != nil {
			e.logger.Error("failed to update host record", zap.Error(err))
		}
		return
	}

	stdout, stderr, exitCode, err := e.runSSH(ctx, hr.Host, record.Account, password, record.Script, record.Args, time.Duration(timeout)*time.Second)

	hr.DurationMs = time.Since(start).Milliseconds()
	hr.Stdout = stdout
	hr.Stderr = stderr
	hr.ExitCode = exitCode

	if err != nil || exitCode != 0 {
		hr.Status = model.TaskStatusFail
		if err != nil {
			hr.Stderr += "\n" + err.Error()
		}
	} else {
		hr.Status = model.TaskStatusSuccess
	}

	if err := e.recRepo.UpdateHostRecord(ctx, hr); err != nil {
		e.logger.Error("failed to update host record",
			zap.Uint("task_id", record.ID),
			zap.String("host", hr.Host),
			zap.Error(err),
		)
	}
}

// loadKnownHosts reads the known_hosts file and returns a ssh.HostKeyCallback.
// If the file does not exist or cannot be parsed, it falls back to
// ssh.InsecureIgnoreHostKey and logs a warning.
func (e *TaskExecutor) loadKnownHosts() ssh.HostKeyCallback {
	callback, err := knownhosts.New(e.knownHostsFile)
	if err != nil {
		e.logger.Warn("SSH known_hosts file unavailable, falling back to insecure host key verification — this is vulnerable to MITM attacks",
			zap.String("path", e.knownHostsFile),
			zap.Error(err),
		)
		return ssh.InsecureIgnoreHostKey()
	}

	e.logger.Info("Loaded SSH known_hosts for host key verification",
		zap.String("path", e.knownHostsFile),
	)
	return callback
}

// runSSH connects to a host via SSH and executes the script.
func (e *TaskExecutor) runSSH(ctx context.Context, host, account, password, script, args string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	// Ensure host has a port
	if _, _, splitErr := net.SplitHostPort(host); splitErr != nil {
		host = host + ":22"
	}

	// Build SSH config
	config := &ssh.ClientConfig{
		User: account,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: e.loadKnownHosts(),
		Timeout:         10 * time.Second,
	}

	// Connect
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", "", -1, fmt.Errorf("SSH dial failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("SSH session failed: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Build command. The script itself is an admin-authored shell script executed
	// by design (gated behind the task.execute RBAC permission); it is intentionally
	// run through the remote shell. User-supplied args, however, are POSIX single-quoted
	// per token so they are passed as literal arguments and cannot inject shell syntax.
	cmd := script
	if strings.TrimSpace(args) != "" {
		cmd = cmd + " " + quoteSSHArgs(args)
	}

	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		// Wait for the goroutine to finish writing to buffers before reading.
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
		return stdoutBuf.String(), stderrBuf.String(), -1, ctx.Err()
	case runErr := <-done:
		if runErr != nil {
			if exitErr, ok := runErr.(*ssh.ExitError); ok {
				return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitStatus(), nil
			}
			return stdoutBuf.String(), stderrBuf.String(), -1, runErr
		}
		return stdoutBuf.String(), stderrBuf.String(), 0, nil
	case <-time.After(timeout):
		_ = session.Signal(ssh.SIGKILL)
		// Wait for the goroutine to finish writing to buffers before reading.
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
		return stdoutBuf.String(), stderrBuf.String(), -1, fmt.Errorf("execution timed out after %s", timeout)
	}
}

// quoteSSHArgs splits a free-form args string on whitespace and POSIX single-quotes
// each token. The result is safe to append after a script on the remote shell: every
// token is passed as a single literal argument, so shell metacharacters (; | & $() ` >
// {} etc.) in user input are inert. This replaces the previous blocklist, which was
// trivially bypassable (e.g. $VAR, ~, * were not blocked).
func quoteSSHArgs(args string) string {
	fields := strings.Fields(args)
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = shellQuote(f)
	}
	return strings.Join(quoted, " ")
}

// shellQuote wraps s in single quotes for POSIX shells, escaping any embedded single
// quote as the standard '\” sequence. The output is a single safe shell word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
