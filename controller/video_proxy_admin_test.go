package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoProxyAdminTestDB(t *testing.T) {
	t.Helper()
	originalDB, originalLogDB := model.DB, model.LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.User{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
	})
}

func videoProxyRequest(taskID string, userID, role int) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/videos/"+taskID+"/content",
		nil,
	)
	context.Params = gin.Params{{Key: "task_id", Value: taskID}}
	context.Set("id", userID)
	context.Set("role", role)
	VideoProxy(context)
	return recorder
}

func TestVideoProxyAllowsCrossUserAdminPreviewButNotOrdinaryUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoProxyAdminTestDB(t)
	dir := t.TempDir()
	withSilkRoadLocalDir(t, dir)

	task := model.Task{
		TaskID: "task_cross_user_preview",
		UserId: 42,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}
	require.NoError(t, model.DB.Create(&task).Error)
	payload := []byte("stored-video")
	_, _, err := service.WriteSilkRoadVideoFile(task.TaskID, bytes.NewReader(payload))
	require.NoError(t, err)

	ordinary := videoProxyRequest(task.TaskID, 7, common.RoleCommonUser)
	assert.Equal(t, http.StatusNotFound, ordinary.Code)

	admin := videoProxyRequest(task.TaskID, 7, common.RoleAdminUser)
	assert.Equal(t, http.StatusOK, admin.Code)
	assert.Equal(t, payload, admin.Body.Bytes())

	var audit model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeManage).First(&audit).Error)
	assert.Contains(t, audit.Other, "video.preview")
	assert.Contains(t, audit.Other, task.TaskID)
}

func TestVideoProxyDeniesExpiredVideoToAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupVideoProxyAdminTestDB(t)
	withSilkRoadLocalDir(t, t.TempDir())

	task := model.Task{
		TaskID: "task_admin_expired_preview",
		UserId: 42,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Add(-time.Second).Unix(),
		},
	}
	require.NoError(t, model.DB.Create(&task).Error)

	admin := videoProxyRequest(task.TaskID, 7, common.RoleRootUser)
	assert.Equal(t, http.StatusGone, admin.Code)
}
