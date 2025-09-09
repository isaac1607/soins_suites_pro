package controllers

import (
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"soins-suite-core/internal/modules/tir/tir-etablissement/dto"
	"soins-suite-core/internal/modules/tir/tir-etablissement/services"
)

// TIRLicenseController contrôleur pour gestion licences par admin TIR
type TIRLicenseController struct {
	service *services.TIRLicenseService
}

// NewTIRLicenseController constructeur Fx compatible
func NewTIRLicenseController(service *services.TIRLicenseService) *TIRLicenseController {
	return &TIRLicenseController{
		service: service,
	}
}

// CreateLicense - POST /api/v1/tir/licenses
func (c *TIRLicenseController) CreateLicense(ctx *gin.Context) {
	var req dto.CreateLicenseTIRRequest

	// Validation JSON request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données de création licence invalides",
			"details": map[string]interface{}{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Récupérer informations admin TIR depuis middleware
	adminIDStr := ctx.GetString("tir_admin_id")
	if adminIDStr == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Session admin TIR non trouvée",
		})
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "ID admin TIR invalide",
			"details": map[string]interface{}{
				"admin_id": adminIDStr,
			},
		})
		return
	}

	// Construire informations admin pour traçabilité
	adminInfo := dto.AdminCreationInfo{
		AdminID:     adminIDStr,
		Identifiant: ctx.GetString("tir_identifiant"),
		NiveauAdmin: ctx.GetString("tir_niveau_admin"),
	}

	// Récupérer IP utilisateur pour historique
	var userIP *net.IP
	if clientIP := ctx.ClientIP(); clientIP != "" {
		if parsedIP := net.ParseIP(clientIP); parsedIP != nil {
			userIP = &parsedIP
		}
	}

	// Appel service de création
	result, err := c.service.CreateLicense(
		ctx.Request.Context(),
		req,
		adminID,
		adminInfo,
		userIP,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec création licence",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
		"message": "Licence créée avec succès par admin TIR",
	})
}

// GetLicense - GET /api/v1/tir/licenses/:id
func (c *TIRLicenseController) GetLicense(ctx *gin.Context) {
	// Récupérer ID licence depuis URL
	licenseIDStr := ctx.Param("id")
	licenseID, err := uuid.Parse(licenseIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "ID licence invalide",
			"details": map[string]interface{}{
				"license_id": licenseIDStr,
			},
		})
		return
	}

	// Paramètre optionnel pour inclure l'historique
	includeHistory, _ := strconv.ParseBool(ctx.DefaultQuery("include_history", "false"))

	// Construire informations admin pour traçabilité
	adminInfo := &dto.AdminCreationInfo{
		AdminID:     ctx.GetString("tir_admin_id"),
		Identifiant: ctx.GetString("tir_identifiant"),
		NiveauAdmin: ctx.GetString("tir_niveau_admin"),
	}

	// Appel service
	result, err := c.service.GetLicenseByID(ctx.Request.Context(), licenseID, includeHistory, adminInfo)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Licence non trouvée",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetActiveLicenseByEstablishment - GET /api/v1/tir/establishments/:establishment_id/license/active
func (c *TIRLicenseController) GetActiveLicenseByEstablishment(ctx *gin.Context) {
	// Récupérer ID établissement depuis URL
	establishmentIDStr := ctx.Param("establishment_id")
	establishmentID, err := uuid.Parse(establishmentIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "ID établissement invalide",
			"details": map[string]interface{}{
				"establishment_id": establishmentIDStr,
			},
		})
		return
	}

	// Paramètre optionnel pour inclure l'historique
	includeHistory, _ := strconv.ParseBool(ctx.DefaultQuery("include_history", "false"))

	// Construire informations admin pour traçabilité
	adminInfo := &dto.AdminCreationInfo{
		AdminID:     ctx.GetString("tir_admin_id"),
		Identifiant: ctx.GetString("tir_identifiant"),
		NiveauAdmin: ctx.GetString("tir_niveau_admin"),
	}

	// Appel service
	result, err := c.service.GetActiveLicenseByEstablishment(ctx.Request.Context(), establishmentID, includeHistory, adminInfo)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Aucune licence active trouvée pour cet établissement",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetLicenseListByEstablishment - GET /api/v1/tir/establishments/:establishment_id/licenses
func (c *TIRLicenseController) GetLicenseListByEstablishment(ctx *gin.Context) {
	// Récupérer ID établissement depuis URL
	establishmentIDStr := ctx.Param("establishment_id")
	establishmentID, err := uuid.Parse(establishmentIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "ID établissement invalide",
			"details": map[string]interface{}{
				"establishment_id": establishmentIDStr,
			},
		})
		return
	}

	// Appel service
	result, err := c.service.GetLicenseListByEstablishment(ctx.Request.Context(), establishmentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec récupération liste licences",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetAvailableModules - GET /api/v1/tir/licenses/available-modules
func (c *TIRLicenseController) GetAvailableModules(ctx *gin.Context) {
	// Appel service
	result, err := c.service.GetAvailableFrontOfficeModules(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec récupération modules disponibles",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}