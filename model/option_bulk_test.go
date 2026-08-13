package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateOptionsBulkRollsBackWholeRevision(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:option-bulk-rollback?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	require.NoError(t, db.Create(&[]Option{
		{Key: "brioi_setting.video_tool_groups", Value: `["old"]`},
		{Key: "brioi_setting.profiles", Value: `[{"model":"old"}]`},
	}).Error)
	require.NoError(t, db.Exec(`
		CREATE TRIGGER fail_brioi_profile_update
		BEFORE UPDATE ON options
		WHEN NEW.key = 'brioi_setting.profiles'
		BEGIN
			SELECT RAISE(FAIL, 'forced provider revision failure');
		END
	`).Error)

	err = UpdateOptionsBulk(map[string]string{
		"brioi_setting.video_tool_groups": `["new"]`,
		"brioi_setting.profiles":          `[{"model":"new"}]`,
	})
	require.Error(t, err)

	var groups Option
	var profiles Option
	require.NoError(t, db.Where("key = ?", "brioi_setting.video_tool_groups").First(&groups).Error)
	require.NoError(t, db.Where("key = ?", "brioi_setting.profiles").First(&profiles).Error)
	assert.Equal(t, `["old"]`, groups.Value)
	assert.Equal(t, `[{"model":"old"}]`, profiles.Value)
}
