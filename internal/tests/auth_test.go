// internal/tests/auth_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/javajoker/imi-backend/internal/handlers"
)

type AuthTestSuite struct {
	suite.Suite
	db          *gorm.DB
	router      *gin.Engine
	authHandler *handlers.AuthHandler
}

func (suite *AuthTestSuite) SetupSuite() {
	// Setup test database and router
	// This would typically use a test database
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Initialize test services and handlers
	// authService := services.NewAuthService(suite.db, testConfig)
	// suite.authHandler = handlers.NewAuthHandler(authService)

	// Setup routes
	auth := suite.router.Group("/auth")
	{
		auth.POST("/register", suite.authHandler.Register)
		auth.POST("/login", suite.authHandler.Login)
	}
}

func (suite *AuthTestSuite) TestUserRegistration() {
	// Test successful registration
	registerData := map[string]interface{}{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "TestPass123!",
		"user_type": "creator",
	}

	jsonData, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
}

func (suite *AuthTestSuite) TestUserLogin() {
	// First register a user
	// ... registration code ...

	// Test login
	loginData := map[string]interface{}{
		"email":    "test@example.com",
		"password": "TestPass123!",
	}

	jsonData, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
