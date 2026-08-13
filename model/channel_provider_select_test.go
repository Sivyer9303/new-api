package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetRandomSatisfiedChannelForChannelTypeCannotCrossProvider(t *testing.T) {
	previousDB := DB
	previousMemoryCache := common.MemoryCacheEnabled
	t.Cleanup(func() {
		DB = previousDB
		common.MemoryCacheEnabled = previousMemoryCache
	})

	for _, memoryCache := range []bool{false, true} {
		t.Run(fmt.Sprintf("memory_cache_%t", memoryCache), func(t *testing.T) {
			db, err := gorm.Open(
				sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())),
				&gorm.Config{},
			)
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			DB = db
			common.MemoryCacheEnabled = memoryCache

			highPriority := int64(100)
			lowPriority := int64(10)
			require.NoError(t, db.Create(&[]Channel{
				{
					Id:       951,
					Type:     constant.ChannelTypeSilkRoad,
					Name:     "silkroad-provider",
					Key:      "silkroad-key",
					Status:   common.ChannelStatusEnabled,
					Group:    "video",
					Models:   "same-model",
					Priority: &highPriority,
				},
				{
					Id:       952,
					Type:     constant.ChannelTypeBrioi,
					Name:     "brioi-provider",
					Key:      "brioi-key",
					Status:   common.ChannelStatusEnabled,
					Group:    "video",
					Models:   "same-model",
					Priority: &lowPriority,
				},
			}).Error)
			require.NoError(t, db.Create(&[]Ability{
				{
					Group:     "video",
					Model:     "same-model",
					ChannelId: 951,
					Enabled:   true,
					Priority:  &highPriority,
				},
				{
					Group:     "video",
					Model:     "same-model",
					ChannelId: 952,
					Enabled:   true,
					Priority:  &lowPriority,
				},
			}).Error)
			if memoryCache {
				InitChannelCache()
			}

			selected, err := GetRandomSatisfiedChannelForChannelType(
				"video",
				"same-model",
				0,
				"/v1/video/generations",
				constant.ChannelTypeBrioi,
			)
			require.NoError(t, err)
			require.NotNil(t, selected)
			assert.Equal(t, constant.ChannelTypeBrioi, selected.Type)

			require.NoError(t, db.Model(&Channel{}).
				Where("id = ?", 952).
				Update("status", common.ChannelStatusManuallyDisabled).Error)
			if memoryCache {
				InitChannelCache()
			}

			selected, err = GetRandomSatisfiedChannelForChannelType(
				"video",
				"same-model",
				0,
				"/v1/video/generations",
				constant.ChannelTypeBrioi,
			)
			require.NoError(t, err)
			assert.Nil(t, selected)
		})
	}
}
