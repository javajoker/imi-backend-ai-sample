// internal/handlers/verification.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javajoker/imi-backend/internal/services"
	"github.com/javajoker/imi-backend/internal/utils"
)

type VerificationHandler struct {
	authorizationService *services.AuthorizationService
}

func NewVerificationHandler(authorizationService *services.AuthorizationService) *VerificationHandler {
	return &VerificationHandler{
		authorizationService: authorizationService,
	}
}

// GET /verify/:code
func (h *VerificationHandler) VerifyProductByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		utils.BadRequestResponse(c, "Verification code is required", nil)
		return
	}

	// Verify product by code
	authChain, err := h.authorizationService.VerifyProductByCode(code)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"verified":            true,
		"product":             authChain.Product,
		"ip_asset":            authChain.IPAsset,
		"license":             authChain.License,
		"authorization_chain": authChain,
	})
}

// GET /verify/chain/:id
func (h *VerificationHandler) VerifyAuthorizationChain(c *gin.Context) {
	idStr := c.Param("id")
	authChainID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid authorization chain ID", nil)
		return
	}

	// Verify authorization chain
	valid, err := h.authorizationService.VerifyChain(authChainID)
	if err != nil {
		utils.BadRequestResponse(c, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"valid": valid,
	})
}

// GET /verify/chain/:id/history
func (h *VerificationHandler) GetAuthorizationChainHistory(c *gin.Context) {
	idStr := c.Param("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid product ID", nil)
		return
	}

	// Get authorization chain history
	chains, err := h.authorizationService.GetAuthChainHistory(productID)
	if err != nil {
		utils.InternalErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"authorization_chains": chains,
	})
}
