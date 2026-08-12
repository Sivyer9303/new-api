package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestVideoRecoveryRoutesRequireAdministratorAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	requests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/task/video/task_123/diagnostics"},
		{http.MethodPost, "/api/task/video/task_123/storage/retry"},
		{http.MethodPost, "/api/task/video/task_123/provider/confirm"},
		{http.MethodPost, "/api/task/video/task_123/refund"},
	}
	for _, item := range requests {
		t.Run(item.method+" "+item.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				item.method,
				item.path,
				strings.NewReader(`{"reason":"test"}`),
			)
			request.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}
