package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
)

func setupAlertForwarderService(t *testing.T) (*service.AlertForwarderService, *gorm.DB) {
	t.Helper()
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)
	// Ensure table exists
	db.Exec(`CREATE TABLE IF NOT EXISTS alert_forwarders (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(128) NOT NULL UNIQUE,
		description VARCHAR(512) NOT NULL DEFAULT '',
		enabled TINYINT(1) NOT NULL DEFAULT 1,
		direction VARCHAR(32) NOT NULL,
		priority INT NOT NULL DEFAULT 0,
		inbound_config JSON DEFAULT NULL,
		outbound_config JSON DEFAULT NULL,
		inbound_severity_mapping JSON DEFAULT NULL,
		outbound_severity_mapping JSON DEFAULT NULL,
		platform_capabilities JSON DEFAULT NULL,
		match_labels JSON DEFAULT NULL,
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	db.Exec("DELETE FROM alert_forwarders")

	forwarderRepo := repository.NewAlertForwarderRepository(db)
	mediaRepo := repository.NewNotifyMediaRepository(db)
	svc := service.NewAlertForwarderService(forwarderRepo, mediaRepo, nil, testutil.TestLogger())
	return svc, db
}

// ===== Create Tests =====

func TestAlertForwarder_Create_Inbound_Integrate(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-inbound-integrate",
		Enabled:   true,
		Direction: model.ForwarderDirectionInbound,
		InboundConfig: &model.InboundConfig{
			SourceFormat: model.SourceFormatAlertmanager,
			Mode:         model.InboundModeIntegrate,
			AuthType:     model.ForwarderAuthNone,
		},
	}

	err := svc.Create(context.Background(), fwd)
	require.NoError(t, err)
	assert.NotZero(t, fwd.ID)
	assert.NotNil(t, fwd.PlatformCapabilities)
	assert.True(t, fwd.PlatformCapabilities.EnableNotification)
}

func TestAlertForwarder_Create_Inbound_Proxy(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-inbound-proxy",
		Enabled:   true,
		Direction: model.ForwarderDirectionInbound,
		InboundConfig: &model.InboundConfig{
			SourceFormat: model.SourceFormatAlertmanager,
			Mode:         model.InboundModeProxy,
			AuthType:     model.ForwarderAuthNone,
			ProxyTarget: &model.OutboundConfig{
				TargetURL: "https://external.example.com/webhook",
			},
		},
	}

	err := svc.Create(context.Background(), fwd)
	require.NoError(t, err)
	assert.NotZero(t, fwd.ID)
}

func TestAlertForwarder_Create_Outbound(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-outbound",
		Enabled:   true,
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://external.example.com/webhook",
		},
	}

	err := svc.Create(context.Background(), fwd)
	require.NoError(t, err)
	assert.NotZero(t, fwd.ID)
}

func TestAlertForwarder_Create_Bidirectional(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-bidirectional",
		Enabled:   true,
		Direction: model.ForwarderDirectionBidirectional,
		InboundConfig: &model.InboundConfig{
			SourceFormat: model.SourceFormatAlertmanager,
			Mode:         model.InboundModeIntegrate,
			AuthType:     model.ForwarderAuthNone,
		},
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://external.example.com/webhook",
		},
	}

	err := svc.Create(context.Background(), fwd)
	require.NoError(t, err)
}

// ===== Validation Tests =====

func TestAlertForwarder_Create_InvalidDirection(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-invalid",
		Direction: "invalid",
	}

	err := svc.Create(context.Background(), fwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid direction")
}

func TestAlertForwarder_Create_Inbound_MissingConfig(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-missing-config",
		Direction: model.ForwarderDirectionInbound,
	}

	err := svc.Create(context.Background(), fwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inbound_config is required")
}

func TestAlertForwarder_Create_Outbound_MissingConfig(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-missing-outbound",
		Direction: model.ForwarderDirectionOutbound,
	}

	err := svc.Create(context.Background(), fwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outbound_config is required")
}

func TestAlertForwarder_Create_Outbound_NoTarget(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-no-target",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			Method: "POST",
		},
	}

	err := svc.Create(context.Background(), fwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target_media_id or target_url is required")
}

func TestAlertForwarder_Create_Proxy_MissingTarget(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-proxy-no-target",
		Direction: model.ForwarderDirectionInbound,
		InboundConfig: &model.InboundConfig{
			SourceFormat: model.SourceFormatAlertmanager,
			Mode:         model.InboundModeProxy,
			AuthType:     model.ForwarderAuthNone,
		},
	}

	err := svc.Create(context.Background(), fwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proxy_target is required")
}

func TestAlertForwarder_Create_DuplicateName(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd1 := &model.AlertForwarder{
		Name:      "duplicate-name",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com/1",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd1))

	fwd2 := &model.AlertForwarder{
		Name:      "duplicate-name",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com/2",
		},
	}
	err := svc.Create(context.Background(), fwd2)
	assert.Error(t, err)
}

// ===== CRUD Tests =====

func TestAlertForwarder_GetByID(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-get",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd))

	found, err := svc.GetByID(context.Background(), fwd.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-get", found.Name)
}

func TestAlertForwarder_GetByID_NotFound(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	_, err := svc.GetByID(context.Background(), 99999)
	assert.Error(t, err)
}

func TestAlertForwarder_List_Pagination(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	for i := 0; i < 5; i++ {
		fwd := &model.AlertForwarder{
			Name:      fmt.Sprintf("list-test-%d", i),
			Direction: model.ForwarderDirectionOutbound,
			OutboundConfig: &model.OutboundConfig{
				TargetURL: fmt.Sprintf("https://example.com/%d", i),
			},
		}
		require.NoError(t, svc.Create(context.Background(), fwd))
	}

	list, total, err := svc.List(context.Background(), 1, 3, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, list, 3)

	list2, total2, err2 := svc.List(context.Background(), 2, 3, "", nil)
	require.NoError(t, err2)
	assert.Equal(t, int64(5), total2)
	assert.Len(t, list2, 2)
}

func TestAlertForwarder_List_FilterDirection(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	require.NoError(t, svc.Create(context.Background(), &model.AlertForwarder{
		Name: "inbound-1", Direction: model.ForwarderDirectionInbound,
		InboundConfig: &model.InboundConfig{SourceFormat: model.SourceFormatAlertmanager, Mode: model.InboundModeIntegrate, AuthType: model.ForwarderAuthNone},
	}))
	require.NoError(t, svc.Create(context.Background(), &model.AlertForwarder{
		Name: "outbound-1", Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{TargetURL: "https://example.com"},
	}))

	list, total, err := svc.List(context.Background(), 1, 10, "inbound", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
	assert.Equal(t, "inbound-1", list[0].Name)
}

func TestAlertForwarder_List_FilterEnabled(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	require.NoError(t, svc.Create(context.Background(), &model.AlertForwarder{
		Name: "enabled-1", Enabled: true, Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{TargetURL: "https://example.com"},
	}))
	// Create then disable (GORM default:true overrides zero value on create)
	fwd2 := &model.AlertForwarder{
		Name: "disabled-1", Enabled: true, Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{TargetURL: "https://example.com"},
	}
	require.NoError(t, svc.Create(context.Background(), fwd2))
	require.NoError(t, svc.Disable(context.Background(), fwd2.ID))

	enabled := true
	list, total, err := svc.List(context.Background(), 1, 10, "", &enabled)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
	assert.Equal(t, "enabled-1", list[0].Name)
}

func TestAlertForwarder_Update(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "original-name",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd))

	fwd.Name = "updated-name"
	fwd.OutboundConfig.TargetURL = "https://updated.com"
	err := svc.Update(context.Background(), fwd)
	require.NoError(t, err)

	found, err := svc.GetByID(context.Background(), fwd.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", found.Name)
}

func TestAlertForwarder_Update_InvalidDirection(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "test-update-invalid",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd))

	fwd.Direction = "invalid"
	err := svc.Update(context.Background(), fwd)
	assert.Error(t, err)
}

func TestAlertForwarder_Delete(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "to-delete",
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd))

	err := svc.Delete(context.Background(), fwd.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), fwd.ID)
	assert.Error(t, err)
}

// ===== Enable/Disable Tests =====

func TestAlertForwarder_EnableDisable(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	fwd := &model.AlertForwarder{
		Name:      "toggle-test",
		Enabled:   true,
		Direction: model.ForwarderDirectionOutbound,
		OutboundConfig: &model.OutboundConfig{
			TargetURL: "https://example.com",
		},
	}
	require.NoError(t, svc.Create(context.Background(), fwd))

	// Disable
	err := svc.Disable(context.Background(), fwd.ID)
	require.NoError(t, err)
	found, _ := svc.GetByID(context.Background(), fwd.ID)
	assert.False(t, found.Enabled)

	// Enable
	err = svc.Enable(context.Background(), fwd.ID)
	require.NoError(t, err)
	found, _ = svc.GetByID(context.Background(), fwd.ID)
	assert.True(t, found.Enabled)
}

// ===== Batch Operations Tests =====

func TestAlertForwarder_BatchEnable(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	ids := make([]uint, 3)
	for i := 0; i < 3; i++ {
		fwd := &model.AlertForwarder{
			Name:      fmt.Sprintf("batch-%d", i),
			Enabled:   false,
			Direction: model.ForwarderDirectionOutbound,
			OutboundConfig: &model.OutboundConfig{
				TargetURL: fmt.Sprintf("https://example.com/%d", i),
			},
		}
		require.NoError(t, svc.Create(context.Background(), fwd))
		ids[i] = fwd.ID
	}

	err := svc.BatchEnable(context.Background(), ids)
	require.NoError(t, err)

	for _, id := range ids {
		found, _ := svc.GetByID(context.Background(), id)
		assert.True(t, found.Enabled)
	}
}

func TestAlertForwarder_BatchDelete(t *testing.T) {
	svc, _ := setupAlertForwarderService(t)

	ids := make([]uint, 3)
	for i := 0; i < 3; i++ {
		fwd := &model.AlertForwarder{
			Name:      fmt.Sprintf("batch-del-%d", i),
			Direction: model.ForwarderDirectionOutbound,
			OutboundConfig: &model.OutboundConfig{
				TargetURL: fmt.Sprintf("https://example.com/%d", i),
			},
		}
		require.NoError(t, svc.Create(context.Background(), fwd))
		ids[i] = fwd.ID
	}

	err := svc.BatchDelete(context.Background(), ids)
	require.NoError(t, err)

	for _, id := range ids {
		_, err := svc.GetByID(context.Background(), id)
		assert.Error(t, err)
	}
}

// ===== Severity Mapping Tests =====

func TestSeverityMapping_ApplySeverityMapping(t *testing.T) {
	tests := []struct {
		name     string
		config   *model.SeverityMappingConfig
		input    string
		expected string
		applied  bool
	}{
		{
			name:     "nil config",
			config:   nil,
			input:    "critical",
			expected: "critical",
			applied:  false,
		},
		{
			name:     "disabled",
			config:   &model.SeverityMappingConfig{Enabled: false},
			input:    "critical",
			expected: "critical",
			applied:  false,
		},
		{
			name: "exact match",
			config: &model.SeverityMappingConfig{
				Enabled: true,
				Mapping: map[string]string{"critical": "P0", "warning": "P2"},
			},
			input:    "critical",
			expected: "P0",
			applied:  true,
		},
		{
			name: "fallback to default",
			config: &model.SeverityMappingConfig{
				Enabled:         true,
				Mapping:         map[string]string{"critical": "P0"},
				DefaultSeverity: "P3",
			},
			input:    "unknown",
			expected: "P3",
			applied:  true,
		},
		{
			name: "no match no default",
			config: &model.SeverityMappingConfig{
				Enabled: true,
				Mapping: map[string]string{"critical": "P0"},
			},
			input:    "unknown",
			expected: "unknown",
			applied:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applied := tt.config.ApplySeverityMapping(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.applied, applied)
		})
	}
}

// ===== SanitizeForResponse Tests =====

func TestAlertForwarder_SanitizeForResponse(t *testing.T) {
	fwd := &model.AlertForwarder{
		Name: "test",
		InboundConfig: &model.InboundConfig{
			AuthConfig: &model.AuthConfig{
				Token:      "secret-token",
				Password:   "secret-password",
				HMACSecret: "secret-hmac",
			},
		},
	}

	sanitized := fwd.SanitizeForResponse()
	assert.Equal(t, "***", sanitized.InboundConfig.AuthConfig.Token)
	assert.Equal(t, "***", sanitized.InboundConfig.AuthConfig.Password)
	assert.Equal(t, "***", sanitized.InboundConfig.AuthConfig.HMACSecret)
	// Original should be unchanged
	assert.Equal(t, "secret-token", fwd.InboundConfig.AuthConfig.Token)
}

func TestAlertForwarder_SanitizeForResponse_NilAuth(t *testing.T) {
	fwd := &model.AlertForwarder{
		Name:          "test",
		InboundConfig: &model.InboundConfig{},
	}

	sanitized := fwd.SanitizeForResponse()
	assert.Nil(t, sanitized.InboundConfig.AuthConfig)
}
