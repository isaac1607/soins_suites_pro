package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"soins-suite-core/internal/modules/tir/tir-etablissement/dto"
	"soins-suite-core/internal/modules/tir/tir-etablissement/services"
)

// TIREstablishmentController contrôleur pour gestion établissements par admin TIR
type TIREstablishmentController struct {
	service *services.TIREstablishmentService
}

// NewTIREstablishmentController constructeur Fx compatible
func NewTIREstablishmentController(service *services.TIREstablishmentService) *TIREstablishmentController {
	return &TIREstablishmentController{
		service: service,
	}
}

// CreateEstablishment - POST /api/v1/tir/establishments
func (c *TIREstablishmentController) CreateEstablishment(ctx *gin.Context) {
	var req dto.CreateEstablishmentTIRRequest

	// Validation JSON request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données de création établissement invalides",
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

	// Appel service de création
	result, err := c.service.CreateEstablishment(
		ctx.Request.Context(),
		req,
		adminID,
		adminInfo,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec création établissement",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
		"message": "Établissement créé avec succès par admin TIR",
	})
}

// GetEstablishment - GET /api/v1/tir/establishments/:id
func (c *TIREstablishmentController) GetEstablishment(ctx *gin.Context) {
	// Récupérer ID établissement depuis URL
	establishmentIDStr := ctx.Param("id")
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
	establishment, err := c.service.GetEstablishmentByID(ctx.Request.Context(), establishmentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Établissement non trouvé",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    establishment,
	})
}

// GetEstablishmentByCode - GET /api/v1/tir/establishments/by-code/:code
func (c *TIREstablishmentController) GetEstablishmentByCode(ctx *gin.Context) {
	// Récupérer code établissement depuis URL
	code := ctx.Param("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Code établissement requis",
		})
		return
	}

	// Appel service
	establishment, err := c.service.GetEstablishmentByCode(ctx.Request.Context(), code)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Établissement non trouvé",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    establishment,
	})
}

// UpdateEstablishment - PUT /api/v1/tir/establishments/:id
func (c *TIREstablishmentController) UpdateEstablishment(ctx *gin.Context) {
	// Récupérer ID établissement depuis URL
	establishmentIDStr := ctx.Param("id")
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

	var req dto.UpdateEstablishmentTIRRequest

	// Validation JSON request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données de mise à jour établissement invalides",
			"details": map[string]interface{}{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Récupérer informations admin TIR depuis middleware
	adminIDStr := ctx.GetString("tir_admin_id")
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "ID admin TIR invalide",
		})
		return
	}

	// Construire informations admin pour traçabilité
	adminInfo := dto.AdminCreationInfo{
		AdminID:     adminIDStr,
		Identifiant: ctx.GetString("tir_identifiant"),
		NiveauAdmin: ctx.GetString("tir_niveau_admin"),
	}

	// Appel service de mise à jour
	result, err := c.service.UpdateEstablishmentByAdminTir(
		ctx.Request.Context(),
		establishmentID,
		req,
		adminID,
		adminInfo,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Échec mise à jour établissement",
			"details": map[string]interface{}{
				"reason": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Établissement mis à jour avec succès par admin TIR",
	})
}