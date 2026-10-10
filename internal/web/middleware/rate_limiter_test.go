package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetCount(ctx context.Context, token string) (int, error) {
	args := m.Called(ctx, token)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabase) IncrementCount(ctx context.Context, token string, duration time.Duration) error {
	return m.Called(ctx, token, duration).Error(0)
}

func (m *MockDatabase) UpdateExpiration(ctx context.Context, token string, duration time.Duration) error {
	return m.Called(ctx, token, duration).Error(0)
}

func TestAddTokenConf(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)

	rl.AddTokenConf("apiToken", 100, 10*time.Second)

	assert.Equal(t, 100, rl.Conf["apiToken"].Limit)
	assert.Equal(t, 10*time.Second, rl.Conf["apiToken"].Interval)
}

func TestCheckRateLimitCount_BelowLimit(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)

	db.On("GetCount", mock.Anything, "apiToken").Return(3, nil)

	ok, err := rl.CheckRateLimitCount(context.Background(), "apiToken", false)

	assert.NoError(t, err)
	assert.True(t, ok)
	db.AssertExpectations(t)
}

func TestCheckRateLimitCount_AboveLimit(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)

	db.On("GetCount", mock.Anything, "apiToken").Return(5, nil)

	ok, err := rl.CheckRateLimitCount(context.Background(), "apiToken", false)

	assert.NoError(t, err)
	assert.False(t, ok)
	db.AssertExpectations(t)
}

func TestIncreaseRateLimitCount(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)

	db.On("IncrementCount", mock.Anything, "apiToken", 10*time.Second).Return(nil)

	err := rl.IncreaseRateLimitCount(context.Background(), "apiToken", false)
	assert.NoError(t, err)
	db.AssertExpectations(t)
}

func TestUpdateExpiration(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)

	db.On("UpdateExpiration", mock.Anything, "apiToken", 10*time.Second).Return(nil)

	err := rl.UpdateExpiration(context.Background(), "apiToken", false)
	assert.NoError(t, err)
	db.AssertExpectations(t)
}

func TestRateLimiterCore_FailClosedOnRedisError(t *testing.T) {
	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)
	// FailOpen defaults to false (fail-closed).

	db.On("GetCount", mock.Anything, "apiToken").Return(0, errors.New("redis: connection timeout"))

	ok, err := RateLimiterCore(context.Background(), rl, "apiToken", false)

	assert.Error(t, err)
	assert.False(t, ok)
	db.AssertExpectations(t)
}

func TestRateLimiterMiddleware_FailClosed_Returns500(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("API_KEY", "apiToken")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)
	rl.SetFailOpen(false)

	db.On("GetCount", mock.Anything, "apiToken").Return(0, errors.New("redis: connection timeout"))

	handler := RateLimiterMiddleware(context.Background(), rl)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	db.AssertExpectations(t)
}

func TestRateLimiterMiddleware_FailOpen_PassesThrough(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("API_KEY", "apiToken")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := &MockDatabase{}
	rl := NewRateLimiter(db)
	rl.AddTokenConf("apiToken", 5, 10*time.Second)
	rl.SetFailOpen(true)

	db.On("GetCount", mock.Anything, "apiToken").Return(0, errors.New("redis: connection timeout"))

	handler := RateLimiterMiddleware(context.Background(), rl)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	db.AssertExpectations(t)
}
