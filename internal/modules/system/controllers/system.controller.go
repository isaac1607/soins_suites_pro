package controllers

import (
	"net/http"

	"soins-suite-core/internal/modules/system/dto"
	"soins-suite-core/internal/modules/system/services"
	middlewareAuth "soins-suite-core/internal/shared/middleware/authentication"

	"github.com/gin-gonic/gin"
)

type SystemController struct {
	service *services.SystemService
}

func NewSystemController(service *services.SystemService) *SystemController {
	return &SystemController{
		service: service,
	}
}

// GetSystemInfo - GET /api/v1/system/info
// Récupère toutes les informations système pour l'établissement connecté
func (c *SystemController) GetSystemInfo(ctx *gin.Context) {
	// Récupération de l'établissement depuis le middleware
	establishmentContext, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"middleware_error": "EstablishmentMiddleware n'a pas injecté le contexte",
			},
		})
		return
	}

	// Casting du contexte établissement (type EstablishmentContext du middleware)
	establishment, ok := establishmentContext.(middlewareAuth.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format contexte établissement invalide",
			"details": map[string]interface{}{
				"type_expected": "EstablishmentContext",
				"type_received": "unknown",
			},
		})
		return
	}

	// L'ID est directement disponible dans EstablishmentContext
	establishmentID := establishment.ID

	// Appel du service
	systemInfo, err := c.service.GetSystemInfo(ctx.Request.Context(), establishmentID)
	if err != nil {
		// Erreur technique uniquement (licence manquante est gérée gracieusement)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur récupération informations système",
			"details": map[string]interface{}{
				"establishment_id": establishmentID,
				"error_message":    err.Error(),
			},
		})
		return
	}

	// Génération des alertes
	alertes := c.service.GenerateAlertes(&systemInfo.Licence)

	// Construction de la réponse standard
	response := dto.StandardAPIResponse{
		Success: true,
		Data:    systemInfo,
		Alertes: alertes,
	}

	ctx.JSON(http.StatusOK, response)
}

// GetAuthorizedModules - GET /api/v1/system/modules/authorized
// Récupère uniquement les modules autorisés par la licence pour navigation dynamique
func (c *SystemController) GetAuthorizedModules(ctx *gin.Context) {
	// Récupération de l'établissement depuis le middleware
	establishmentContext, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"middleware_error": "EstablishmentMiddleware n'a pas injecté le contexte",
			},
		})
		return
	}

	// Casting du contexte établissement
	establishment, ok := establishmentContext.(middlewareAuth.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format contexte établissement invalide",
			"details": map[string]interface{}{
				"type_expected": "EstablishmentContext",
				"type_received": "unknown",
			},
		})
		return
	}

	establishmentID := establishment.ID

	// Appel du service
	authorizedModules, err := c.service.GetAuthorizedModulesOnly(ctx.Request.Context(), establishmentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur récupération modules autorisés",
			"details": map[string]interface{}{
				"establishment_id": establishmentID,
				"error_message":    err.Error(),
			},
		})
		return
	}

	// Construction de la réponse standard
	response := dto.StandardAPIResponse{
		Success: true,
		Data:    authorizedModules,
	}

	ctx.JSON(http.StatusOK, response)
}

// ActivateOffline - POST /api/v1/system/activated
// Endpoint pour déclencher la synchronisation depuis le back-office local
func (c *SystemController) ActivateOffline(ctx *gin.Context) {
	// Vérifier le header X-Client-Type
	clientType := ctx.GetHeader("X-Client-Type")
	if clientType != "back-office" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Header X-Client-Type: back-office requis",
			"code":  "INVALID_CLIENT_TYPE",
			"details": map[string]interface{}{
				"header_required": "X-Client-Type: back-office",
			},
		})
		return
	}

	// Récupération de l'établissement depuis le middleware
	establishmentContext, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"middleware_error": "EstablishmentMiddleware n'a pas injecté le contexte",
			},
		})
		return
	}

	establishment, ok := establishmentContext.(middlewareAuth.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format contexte établissement invalide",
			"details": map[string]interface{}{
				"type_expected": "EstablishmentContext",
				"type_received": "unknown",
			},
		})
		return
	}

	// Bind de la requête
	var req dto.ActivateOfflineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données invalides",
			"code":  "INVALID_REQUEST_FORMAT",
			"details": map[string]interface{}{
				"binding_error": err.Error(),
			},
		})
		return
	}

	// Vérifier que le code correspond au contexte
	if req.CodeEtablissement != establishment.Code {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Code établissement ne correspond pas au contexte",
			"code":  "ESTABLISHMENT_CODE_MISMATCH",
			"details": map[string]interface{}{
				"code_in_request": req.CodeEtablissement,
				"code_in_context": establishment.Code,
			},
		})
		return
	}

	// Appel du service
	result, err := c.service.ActivateOfflineSystem(ctx.Request.Context(), req)
	if err != nil {
		// Gestion des erreurs spécifiques
		if serviceErr, ok := err.(*services.ServiceError); ok {
			var statusCode int
			switch serviceErr.Type {
			case "configuration":
				statusCode = http.StatusInternalServerError
			case "not_found":
				statusCode = http.StatusBadRequest
			case "conflict":
				statusCode = http.StatusConflict
			case "network":
				statusCode = http.StatusBadGateway
			default:
				statusCode = http.StatusInternalServerError
			}

			ctx.JSON(statusCode, gin.H{
				"error": serviceErr.Message,
				"code":  serviceErr.Code,
				"details": serviceErr.Details,
			})
			return
		}

		// Erreur générique
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur activation système offline",
			"code":  "ACTIVATION_ERROR",
			"details": map[string]interface{}{
				"error_message": err.Error(),
			},
		})
		return
	}

	// Réponse succès
	response := dto.StandardAPIResponse{
		Success: true,
		Data:    result,
	}

	ctx.JSON(http.StatusOK, response)
}

// SynchronizeOffline - POST /api/v1/system/offline/synchronised
// Endpoint pour synchronisation initiale des données (serveur central -> serveur local)
func (c *SystemController) SynchronizeOffline(ctx *gin.Context) {
	// Récupération de l'établissement depuis le middleware
	establishmentContext, exists := ctx.Get("establishment")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Contexte établissement manquant",
			"details": map[string]interface{}{
				"middleware_error": "EstablishmentMiddleware n'a pas injecté le contexte",
			},
		})
		return
	}

	establishment, ok := establishmentContext.(middlewareAuth.EstablishmentContext)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format contexte établissement invalide",
			"details": map[string]interface{}{
				"type_expected": "EstablishmentContext",
				"type_received": "unknown",
			},
		})
		return
	}

	// Bind de la requête
	var req dto.SyncOfflineRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Données invalides",
			"code":  "INVALID_REQUEST_FORMAT",
			"details": map[string]interface{}{
				"binding_error": err.Error(),
			},
		})
		return
	}

	// Appel du service
	result, err := c.service.SynchronizeOfflineData(ctx.Request.Context(), req, establishment.Code)
	if err != nil {
		// Gestion des erreurs spécifiques
		if serviceErr, ok := err.(*services.ServiceError); ok {
			var statusCode int
			switch serviceErr.Type {
			case "not_found":
				statusCode = http.StatusNotFound
			case "conflict":
				statusCode = http.StatusConflict
			default:
				statusCode = http.StatusInternalServerError
			}

			ctx.JSON(statusCode, gin.H{
				"error": serviceErr.Message,
				"code":  serviceErr.Code,
				"details": serviceErr.Details,
			})
			return
		}

		// Erreur générique
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur synchronisation données",
			"code":  "SYNCHRONIZATION_ERROR",
			"details": map[string]interface{}{
				"error_message": err.Error(),
			},
		})
		return
	}

	// Réponse succès
	response := dto.StandardAPIResponse{
		Success: true,
		Data:    result,
	}

	ctx.JSON(http.StatusOK, response)
}
