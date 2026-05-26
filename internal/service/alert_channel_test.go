package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
)

func setupAlertChannelService(t *testing.T) (*service.AlertChannelService, *gorm.DB) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)
	repo := repository.NewAlertChannelRepository(db)
	mediaRepo := repository.NewNotifyMediaRepository(db)
	svc := service.NewAlertChannelService(repo, mediaRepo, nil, testutil.TestLogger())
	return svc, db
}

func TestAlertChannelService_Create(t *testing.T) {
	svc, db := setupAlertChannelService(t)

	ch := &model.AlertChannel{
		Name:      "test-channel",
		MediaID:   1,
		IsEnabled: true,
	}

	err := svc.Create(context.Background(), ch)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)

	var found model.AlertChannel
	require.NoError(t, db.First(&found, ch.ID).Error)
	assert.Equal(t, "test-channel", found.Name)
}

func TestAlertChannelService_GetByID_NotFound(t *testing.T) {
	svc, _ := setupAlertChannelService(t)

	_, err := svc.GetByID(context.Background(), 99999)
	assert.Error(t, err)
}

func TestAlertChannelService_TestChannel_MediaNotFound(t *testing.T) {
	svc, db := setupAlertChannelService(t)

	ch := &model.AlertChannel{
		Name:      "test",
		MediaID:   99999,
		IsEnabled: true,
	}
	require.NoError(t, db.Create(ch).Error)

	err := svc.TestChannel(context.Background(), ch.ID)
	assert.Error(t, err)
}
