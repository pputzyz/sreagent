package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/pkg/crypto"
	"github.com/sreagent/sreagent/internal/pkg/datasource"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// DatasourceChangeCallback is called when a datasource is updated.
// Implementors can use this to react to endpoint/config changes
// (e.g. force the evaluator to re-sync rules for the affected datasource).
type DatasourceChangeCallback interface {
	OnDatasourceUpdated(dsID uint)
}

// DataSourceQuerier is the interface consumed by cross-cutting services
// (ai_tools, diagnostic_workflow, rule_generator_dryrun) for datasource queries.
type DataSourceQuerier interface {
	QueryDatasource(ctx context.Context, dsID uint, expression string, queryTime time.Time) (*QueryResponse, error)
	QueryRange(ctx context.Context, dsID uint, expression string, start, end time.Time, step string) (*QueryResponse, error)
	QueryLogs(ctx context.Context, dsID uint, params LogQueryParams) (*LogQueryResponse, error)
	ProxyToDatasource(ctx context.Context, dsID uint, path string, params map[string]string) ([]byte, error)
}

// Compile-time check: *DataSourceService satisfies DataSourceQuerier.
var _ DataSourceQuerier = (*DataSourceService)(nil)

type DataSourceService struct {
	repo        *repository.DataSourceRepository
	logger      *zap.Logger
	queryClient *datasource.QueryClient
	ruleCountFn func(ctx context.Context, dsID uint) (int64, error) // P1-11: optional cascade check
	onChange    DatasourceChangeCallback // optional; notified on endpoint/config changes
}

func NewDataSourceService(repo *repository.DataSourceRepository, logger *zap.Logger) *DataSourceService {
	return &DataSourceService{
		repo:        repo,
		logger:      logger,
		queryClient: datasource.NewQueryClient(),
	}
}

// SetRuleCountFn injects the function used to check alert rule references before deletion (P1-11).
func (s *DataSourceService) SetRuleCountFn(fn func(ctx context.Context, dsID uint) (int64, error)) {
	s.ruleCountFn = fn
}

// SetChangeCallback registers a callback that is invoked after a datasource
// endpoint or config is updated. This allows the evaluator to force-sync
// rules that reference the changed datasource.
func (s *DataSourceService) SetChangeCallback(cb DatasourceChangeCallback) {
	s.onChange = cb
}

// decryptAuthConfig decrypts the datasource's AuthConfig if it is encrypted.
// Returns the plaintext config, or an error if decryption fails.
func (s *DataSourceService) decryptAuthConfig(ds *model.DataSource) (string, error) {
	if ds.AuthConfig == "" {
		return "", nil
	}
	if !crypto.IsEncrypted(ds.AuthConfig) {
		return ds.AuthConfig, nil
	}
	plain, err := crypto.DecryptString(ds.AuthConfig)
	if err != nil {
		s.logger.Error("failed to decrypt datasource auth_config",
			zap.Uint("datasource_id", ds.ID),
			zap.Error(err),
		)
		return "", fmt.Errorf("decrypt auth_config for ds=%d: %w", ds.ID, err)
	}
	return plain, nil
}

// validateEndpoint checks that the endpoint URL does not point to a private/loopback IP (SSRF protection).
// H1: Also checks DNS resolution, IPv6-mapped addresses, and cloud metadata endpoints.
func validateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	// Only allow http/https schemes.
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("endpoint scheme must be http or https, got %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("endpoint hostname is empty")
	}

	// Block known dangerous hostnames (cloud metadata endpoints, internal services).
	lowerHost := strings.ToLower(host)
	blockedHosts := []string{
		"localhost",
		"metadata.google.internal",
		"metadata.google.com",
		"metadata.azure.com",
		"metadata.azure.internal",
		"instance-data",
		"169.254.169.254", // AWS/GCP/Azure metadata IP (defense in depth; also blocked by IP check)
		"kubernetes.default.svc",
		"kubernetes.default",
	}
	for _, blocked := range blockedHosts {
		if lowerHost == blocked || strings.HasSuffix(lowerHost, "."+blocked) {
			return fmt.Errorf("endpoint hostname %q is not allowed", host)
		}
	}

	// Check if hostname is a literal IP.
	ip := net.ParseIP(host)
	if ip != nil {
		return validateIP(ip)
	}

	// DNS resolution with timeout: check all resolved IPs.
	resolver := &net.Resolver{}
	resolveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := resolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		// DNS failure is not necessarily an SSRF risk — allow it (will fail at connection time).
		return nil
	}
	for _, addr := range addrs {
		if err := validateIP(addr.IP); err != nil {
			return fmt.Errorf("endpoint hostname %q resolves to blocked IP %s: %w", host, addr.IP, err)
		}
	}
	return nil
}

// validateIP checks that an IP is not loopback or link-local.
// Private IPs (RFC1918) are allowed because datasources commonly run on internal networks.
// Runtime SSRF protection is handled by NewInternalClient in safehttp.
func validateIP(ip net.IP) error {
	if ip.IsLoopback() {
		return fmt.Errorf("loopback IP not allowed")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("link-local IP not allowed")
	}
	// Check IPv4-mapped IPv6 (e.g., ::ffff:127.0.0.1).
	if ip4 := ip.To4(); ip4 != nil {
		if ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
			return fmt.Errorf("IPv4-mapped blocked IP not allowed")
		}
	}
	return nil
}

func (s *DataSourceService) Create(ctx context.Context, ds *model.DataSource) error {
	// Validate endpoint against SSRF
	if ds.Endpoint != "" {
		if err := validateEndpoint(ds.Endpoint); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
		}
	}

	// Check if name already exists
	existing, err := s.repo.GetByName(ctx, ds.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to check datasource name", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if existing != nil {
		return apperr.WithMessage(apperr.ErrDuplicateName, fmt.Sprintf("datasource '%s' already exists", ds.Name))
	}

	// Encrypt AuthConfig before persisting
	if ds.AuthConfig != "" && !crypto.IsEncrypted(ds.AuthConfig) {
		enc, err := crypto.EncryptString(ds.AuthConfig)
		if err != nil {
			s.logger.Error("failed to encrypt datasource auth_config", zap.Error(err))
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
		ds.AuthConfig = enc
	}

	if err := s.repo.Create(ctx, ds); err != nil {
		s.logger.Error("failed to create datasource", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

func (s *DataSourceService) GetByID(ctx context.Context, id uint) (*model.DataSource, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}
	return ds, nil
}

func (s *DataSourceService) List(ctx context.Context, dsType, search string, page, pageSize int) ([]model.DataSource, int64, error) {
	return s.repo.List(ctx, dsType, search, page, pageSize)
}

func (s *DataSourceService) Update(ctx context.Context, ds *model.DataSource) error {
	existing, err := s.repo.GetByID(ctx, ds.ID)
	if err != nil {
		return apperr.ErrDSNotFound
	}

	// P1-2: Prevent changing datasource type (e.g. prometheus→zabbix breaks everything)
	if existing.Type != ds.Type {
		return apperr.WithMessage(apperr.ErrInvalidParam, "datasource type cannot be changed; create a new datasource instead")
	}

	// Validate endpoint against SSRF
	if ds.Endpoint != "" {
		if err := validateEndpoint(ds.Endpoint); err != nil {
			return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
		}
	}

	// Snapshot original values before mutation for change detection.
	oldEndpoint := existing.Endpoint
	oldAuthConfig := existing.AuthConfig

	// Update fields
	existing.Name = ds.Name
	existing.Type = ds.Type
	existing.Endpoint = ds.Endpoint
	existing.Description = ds.Description
	existing.Labels = ds.Labels
	existing.AuthType = ds.AuthType

	// P1-1: Allow clearing AuthConfig when switching to "none" auth
	if ds.AuthType == "none" || ds.AuthType == "" {
		existing.AuthConfig = ""
		existing.AuthType = ds.AuthType
	} else if ds.AuthConfig != "" {
		// Encrypt AuthConfig if not already encrypted
		if !crypto.IsEncrypted(ds.AuthConfig) {
			enc, err := crypto.EncryptString(ds.AuthConfig)
			if err != nil {
				s.logger.Error("failed to encrypt datasource auth_config", zap.Error(err))
				return apperr.Wrap(apperr.ErrDatabase, err)
			}
			existing.AuthConfig = enc
		} else {
			existing.AuthConfig = ds.AuthConfig
		}
	}
	existing.HealthCheckInterval = ds.HealthCheckInterval
	existing.IsEnabled = ds.IsEnabled

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update datasource", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Notify evaluator if endpoint or auth config changed — rules may need
	// to re-query the new endpoint on the next evaluation cycle.
	endpointChanged := oldEndpoint != existing.Endpoint
	authChanged := oldAuthConfig != existing.AuthConfig
	if s.onChange != nil && (endpointChanged || authChanged) {
		s.logger.Info("datasource endpoint/auth changed, notifying evaluator",
			zap.Uint("datasource_id", existing.ID),
			zap.Bool("endpoint_changed", endpointChanged),
			zap.Bool("auth_changed", authChanged),
		)
		s.onChange.OnDatasourceUpdated(existing.ID)
	}

	return nil
}

func (s *DataSourceService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrDSNotFound
	}

	// P1-11: Check cascade dependencies before deletion
	if s.ruleCountFn != nil {
		count, err := s.ruleCountFn(ctx, id)
		if err != nil {
			s.logger.Error("failed to check alert rule references", zap.Error(err))
			return apperr.Wrap(apperr.ErrDatabase, err)
		}
		if count > 0 {
			return apperr.WithMessage(apperr.ErrInvalidParam, fmt.Sprintf("cannot delete: %d alert rules reference this datasource", count))
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete datasource", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	return nil
}

// HealthCheckResult is the richer result returned to API callers.
type HealthCheckResult struct {
	Status    model.DataSourceStatus `json:"status"`
	Message   string                 `json:"message"`
	LatencyMs int64                  `json:"latency_ms"`
	Version   string                 `json:"version,omitempty"`
}

// HealthCheck performs a multi-phase health probe against the datasource.
// It updates the datasource status in the DB and returns the full result.
func (s *DataSourceService) HealthCheck(ctx context.Context, id uint) (*HealthCheckResult, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	checker, err := datasource.NewChecker(string(ds.Type))
	if err != nil {
		s.logger.Warn("unsupported datasource type for health check",
			zap.String("type", string(ds.Type)),
		)
		return &HealthCheckResult{Status: model.DSStatusUnknown, Message: "unsupported datasource type"}, nil
	}

	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	hr := checker.CheckHealth(ctx, ds.Endpoint, ds.AuthType, authConfig)

	status := model.DSStatusHealthy
	if !hr.Healthy {
		status = model.DSStatusUnhealthy
		s.logger.Warn("datasource health check failed",
			zap.String("datasource", ds.Name),
			zap.String("message", hr.Message),
			zap.Int64("latency_ms", hr.LatencyMs),
		)
	} else {
		s.logger.Info("datasource health check passed",
			zap.String("datasource", ds.Name),
			zap.String("version", hr.Version),
			zap.Int64("latency_ms", hr.LatencyMs),
		)
	}

	if err := s.repo.UpdateHealthStatus(ctx, ds.ID, status, hr.Version); err != nil {
		s.logger.Error("failed to persist datasource health status",
			zap.String("datasource", ds.Name),
			zap.Error(err),
		)
	}

	return &HealthCheckResult{
		Status:    status,
		Message:   hr.Message,
		LatencyMs: hr.LatencyMs,
		Version:   hr.Version,
	}, nil
}

// QueryResponse holds the result of a datasource query test.
type QueryResponse struct {
	ResultType string            `json:"result_type"`
	Series     []QuerySeriesItem `json:"series"`
	RawCount   int               `json:"raw_count"`
	Truncated  bool              `json:"truncated,omitempty"` // P2-6: true when series were truncated
}

// QuerySeriesItem represents a single series in the query response.
type QuerySeriesItem struct {
	Labels map[string]string `json:"labels"`
	Values []QueryDataPoint  `json:"values"`
}

// QueryDataPoint represents a single data point in a series.
type QueryDataPoint struct {
	Timestamp int64   `json:"ts"`
	Value     float64 `json:"value"`
}

// QueryDatasource executes an expression against the given datasource for testing.
func (s *DataSourceService) QueryDatasource(ctx context.Context, dsID uint, expression string, queryTime time.Time) (*QueryResponse, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	qc := s.queryClient
	resp := &QueryResponse{}
	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}

	switch ds.Type {
	case model.DSTypePrometheus, model.DSTypeVictoriaMetrics:
		results, err := qc.InstantQuery(ctx, ds.Endpoint, ds.AuthType, authConfig, expression, queryTime)
		if err != nil {
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		resp.ResultType = "vector"
		for _, r := range results {
			item := QuerySeriesItem{Labels: r.Labels}
			for _, v := range r.Values {
				item.Values = append(item.Values, QueryDataPoint{Timestamp: v.Timestamp.UnixMilli(), Value: v.Value})
			}
			resp.Series = append(resp.Series, item)
		}
	case model.DSTypeVictoriaLogs:
		results, err := datasource.VictoriaLogsInstantQuery(ctx, ds.Endpoint, ds.AuthType, authConfig, expression, 0)
		if err != nil {
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		resp.ResultType = "logs"
		if len(results) > 0 && len(results[0].Values) > 0 {
			resp.RawCount = int(results[0].Values[0].Value)
		}
	default:
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "expression testing not supported for "+string(ds.Type))
	}

	// Limit series count
	if len(resp.Series) > 100 {
		resp.Series = resp.Series[:100]
		resp.Truncated = true
	}
	return resp, nil
}

// QueryRange executes a PromQL range query against the given datasource.
func (s *DataSourceService) QueryRange(ctx context.Context, dsID uint, expression string, start, end time.Time, step string) (*QueryResponse, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	switch ds.Type {
	case model.DSTypePrometheus, model.DSTypeVictoriaMetrics:
		// proceed
	default:
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "range query not supported for "+string(ds.Type))
	}

	qc := s.queryClient
	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	results, err := qc.RangeQuery(ctx, ds.Endpoint, ds.AuthType, authConfig, expression, start, end, step)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}

	resp := &QueryResponse{ResultType: "matrix"}
	for _, r := range results {
		item := QuerySeriesItem{Labels: r.Labels}
		for _, v := range r.Values {
			item.Values = append(item.Values, QueryDataPoint{Timestamp: v.Timestamp.UnixMilli(), Value: v.Value})
		}
		resp.Series = append(resp.Series, item)
	}

	// Limit series count
	if len(resp.Series) > 1000 {
		resp.Series = resp.Series[:1000]
		resp.Truncated = true
	}
	return resp, nil
}

// LogQueryResponse holds the result of a log query.
type LogQueryResponse struct {
	Entries   []datasource.LogEntry `json:"entries"`
	Total     int                   `json:"total"`
	Truncated bool                  `json:"truncated"`
}

// LogQueryParams holds parameters for a log query.
type LogQueryParams struct {
	Expression string
	Start      time.Time
	End        time.Time
	Limit      int
	Index      string // Elasticsearch index (required for ES)
	DateField  string // Elasticsearch date field (default "@timestamp")
}

// QueryLogs executes a LogsQL query against a VictoriaLogs datasource and returns log entries.
func (s *DataSourceService) QueryLogs(ctx context.Context, dsID uint, params LogQueryParams) (*LogQueryResponse, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	if ds.Type != model.DSTypeVictoriaLogs && ds.Type != model.DSTypeElasticsearch {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "log query only supported for victorialogs and elasticsearch datasources")
	}

	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}

	switch ds.Type {
	case model.DSTypeVictoriaLogs:
		result, err := datasource.QueryLogs(ctx, ds.Endpoint, ds.AuthType, authConfig, datasource.QueryLogsParams{
			Query: params.Expression,
			Start: params.Start,
			End:   params.End,
			Limit: params.Limit,
		})
		if err != nil {
			s.logger.Error("log query failed",
				zap.String("datasource", ds.Name),
				zap.String("expression", params.Expression),
				zap.Error(err),
			)
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		return &LogQueryResponse{
			Entries:   result.Entries,
			Total:     result.Total,
			Truncated: result.Truncated,
		}, nil

	case model.DSTypeElasticsearch:
		if params.Index == "" {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "index is required for elasticsearch log queries")
		}
		result, err := datasource.ElasticsearchQueryLogs(ctx, ds.Endpoint, ds.AuthType, authConfig, datasource.ElasticsearchQueryLogsParams{
			Index:     params.Index,
			Query:     params.Expression,
			DateField: params.DateField,
			Start:     params.Start,
			End:       params.End,
			Limit:     params.Limit,
		})
		if err != nil {
			s.logger.Error("elasticsearch log query failed",
				zap.String("datasource", ds.Name),
				zap.String("index", params.Index),
				zap.String("expression", params.Expression),
				zap.Error(err),
			)
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		return &LogQueryResponse{
			Entries:   result.Entries,
			Total:     result.Total,
			Truncated: result.Truncated,
		}, nil

	default:
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "log query not supported for "+string(ds.Type))
	}
}

// LogHistogramParams holds parameters for a log histogram query.
type LogHistogramParams struct {
	Expression string
	Start      time.Time
	End        time.Time
	Step       string
	Index      string // Elasticsearch index (required for ES)
	DateField  string // Elasticsearch date field (default "@timestamp")
}

// LogHistogramBucket represents a single time bucket in the histogram.
type LogHistogramBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// LogHistogramResponse holds the result of a log histogram query.
type LogHistogramResponse struct {
	Buckets []LogHistogramBucket `json:"buckets"`
	Total   int64                `json:"total"`
}

// QueryLogHistogram fetches log hit counts over time buckets.
func (s *DataSourceService) QueryLogHistogram(ctx context.Context, dsID uint, params LogHistogramParams) (*LogHistogramResponse, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	if ds.Type != model.DSTypeVictoriaLogs && ds.Type != model.DSTypeElasticsearch {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "log histogram only supported for victorialogs and elasticsearch datasources")
	}

	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}

	switch ds.Type {
	case model.DSTypeVictoriaLogs:
		result, err := datasource.QueryLogHistogram(ctx, ds.Endpoint, ds.AuthType, authConfig, params.Expression, params.Start, params.End, params.Step)
		if err != nil {
			s.logger.Error("log histogram query failed",
				zap.String("datasource", ds.Name),
				zap.String("expression", params.Expression),
				zap.Error(err),
			)
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		buckets := make([]LogHistogramBucket, len(result.Buckets))
		for i, b := range result.Buckets {
			buckets[i] = LogHistogramBucket{Timestamp: b.Timestamp, Count: b.Count}
		}
		return &LogHistogramResponse{Buckets: buckets, Total: result.Total}, nil

	case model.DSTypeElasticsearch:
		if params.Index == "" {
			return nil, apperr.WithMessage(apperr.ErrInvalidParam, "index is required for elasticsearch log histogram")
		}
		result, err := datasource.ElasticsearchQueryHistogram(ctx, ds.Endpoint, ds.AuthType, authConfig, datasource.ElasticsearchQueryHistogramParams{
			Index:     params.Index,
			Query:     params.Expression,
			DateField: params.DateField,
			Start:     params.Start,
			End:       params.End,
			Step:      params.Step,
		})
		if err != nil {
			s.logger.Error("elasticsearch log histogram failed",
				zap.String("datasource", ds.Name),
				zap.String("index", params.Index),
				zap.Error(err),
			)
			return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
		}
		buckets := make([]LogHistogramBucket, len(result.Buckets))
		for i, b := range result.Buckets {
			buckets[i] = LogHistogramBucket{Timestamp: b.Timestamp, Count: b.Count}
		}
		return &LogHistogramResponse{Buckets: buckets, Total: result.Total}, nil

	default:
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "log histogram not supported for "+string(ds.Type))
	}
}

// ProxyToDatasource proxies an HTTP GET request to the target datasource's API.
// Used for label/metric queries to support PromQL autocompletion.
func (s *DataSourceService) ProxyToDatasource(ctx context.Context, dsID uint, path string, params map[string]string) ([]byte, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}

	qc := s.queryClient
	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	return qc.ProxyGet(ctx, ds.Endpoint, ds.AuthType, authConfig, path, params)
}

// GetESIndices returns a list of non-hidden Elasticsearch indices for the given datasource.
// The datasource must be of type elasticsearch.
func (s *DataSourceService) GetESIndices(ctx context.Context, dsID uint) ([]string, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}
	if ds.Type != model.DSTypeElasticsearch {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "es-indices only supported for elasticsearch datasources")
	}
	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	indices, err := datasource.ElasticsearchGetIndices(ctx, ds.Endpoint, ds.AuthType, authConfig)
	if err != nil {
		s.logger.Error("failed to get ES indices", zap.String("datasource", ds.Name), zap.Error(err))
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	return indices, nil
}

// GetESFields returns field names and types for a given Elasticsearch index.
// The datasource must be of type elasticsearch.
func (s *DataSourceService) GetESFields(ctx context.Context, dsID uint, index string) ([]datasource.FieldInfo, error) {
	ds, err := s.repo.GetByID(ctx, dsID)
	if err != nil {
		return nil, apperr.ErrDSNotFound
	}
	if ds.Type != model.DSTypeElasticsearch {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "es-fields only supported for elasticsearch datasources")
	}
	if index == "" {
		return nil, apperr.WithMessage(apperr.ErrInvalidParam, "index query parameter is required")
	}
	authConfig, err := s.decryptAuthConfig(ds)
	if err != nil {
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	fields, err := datasource.ElasticsearchGetFields(ctx, ds.Endpoint, ds.AuthType, authConfig, index)
	if err != nil {
		s.logger.Error("failed to get ES fields", zap.String("datasource", ds.Name), zap.String("index", index), zap.Error(err))
		return nil, apperr.WithMessage(apperr.ErrExternalAPI, err.Error())
	}
	return fields, nil
}
