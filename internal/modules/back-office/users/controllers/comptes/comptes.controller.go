package comptes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	dto "soins-suite-core/internal/modules/back-office/users/dto/comptes"
	services "soins-suite-core/internal/modules/back-office/users/services/comptes"
)

type ComptesController struct {
	service   *services.ComptesService
	validator *validator.Validate
}

func NewComptesController(service *services.ComptesService) *ComptesController {
	return &ComptesController{
		service:   service,
		validator: validator.New(),
	}
}

func (c *ComptesController) CreateUser(ctx *gin.Context) {
	establishmentCode := ctx.GetHeader("X-Establishment-Code")
	if establishmentCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Header X-Establishment-Code requis",
		})
		return
	}

	establishmentID := ctx.GetString("establishment_id")
	if establishmentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Etablissement non identifié",
		})
		return
	}

	createdByUserID := ctx.GetString("user_id")
	if createdByUserID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Utilisateur non identifié",
		})
		return
	}

	var req dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Données invalides",
			"details": map[string]interface{}{
				"code": "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	if err := c.validateRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Erreur de validation",
			"details": err,
		})
		return
	}

	result, err := c.service.CreateUser(ctx.Request.Context(), req, establishmentID, createdByUserID)
	if err != nil {
		if strings.Contains(err.Error(), "existe déjà") {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "Cet identifiant existe déjà",
				"details": map[string]interface{}{
					"code": "DUPLICATE_IDENTIFIER",
					"champs": map[string]string{
						"identifiant": "Cet identifiant existe déjà",
					},
				},
			})
			return
		}

		if strings.Contains(err.Error(), "licence") {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "Modules non autorisés",
				"details": map[string]interface{}{
					"code": "MODULES_NOT_ALLOWED_BY_LICENSE",
					"message": err.Error(),
				},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la création de l'utilisateur",
			"details": map[string]interface{}{
				"code": "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
	})
}

func (c *ComptesController) validateRequest(req dto.CreateUserRequest) *dto.ValidationError {
	err := c.validator.Struct(req)
	if err == nil {
		return nil
	}

	validationError := &dto.ValidationError{
		Code:   "VALIDATION_ERROR",
		Champs: make(map[string]string),
	}

	for _, fieldErr := range err.(validator.ValidationErrors) {
		fieldName := c.getJSONFieldName(fieldErr.Field())
		message := c.getValidationMessage(fieldErr)
		validationError.Champs[fieldName] = message
	}

	if req.EstAdmin && (req.TypeAdmin == nil || *req.TypeAdmin == "") {
		validationError.Champs["type_admin"] = "Requis quand est_admin est true"
	}

	if req.EstTemporaire && req.DateExpiration == nil {
		validationError.Champs["date_expiration"] = "Requise quand est_temporaire est true"
	}

	if len(validationError.Champs) == 0 {
		return nil
	}

	return validationError
}

func (c *ComptesController) getJSONFieldName(fieldName string) string {
	mapping := map[string]string{
		"Identifiant":                "identifiant",
		"Nom":                       "nom",
		"Prenoms":                   "prenoms",
		"Telephone":                 "telephone",
		"Email":                     "email",
		"Password":                  "password",
		"TypeAdmin":                 "type_admin",
		"RoleMetier":                "role_metier",
		"ModulesAttribues":          "modules_attribues",
		"ModuleID":                  "module_id",
		"AccesToutesRubriques":      "acces_toutes_rubriques",
		"RubriquesSpecifiques":      "rubriques_specifiques",
	}

	if jsonName, exists := mapping[fieldName]; exists {
		return jsonName
	}
	return strings.ToLower(fieldName)
}

func (c *ComptesController) getValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "Ce champ est requis"
	case "min":
		return fmt.Sprintf("Doit contenir au moins %s caractères", err.Param())
	case "max":
		return fmt.Sprintf("Doit contenir au maximum %s caractères", err.Param())
	case "email":
		return "Format d'email invalide"
	case "uuid":
		return "Format UUID invalide"
	case "oneof":
		return fmt.Sprintf("Valeur invalide. Valeurs autorisées: %s", err.Param())
	default:
		return "Valeur invalide"
	}
}

func (c *ComptesController) ListUsers(ctx *gin.Context) {
	establishmentCode := ctx.GetHeader("X-Establishment-Code")
	if establishmentCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Header X-Establishment-Code requis",
		})
		return
	}

	establishmentID := ctx.GetString("establishment_id")
	if establishmentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Etablissement non identifié",
		})
		return
	}

	var query dto.ListUsersQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Paramètres de requête invalides",
			"details": map[string]interface{}{
				"code":    "INVALID_QUERY_PARAMS",
				"message": err.Error(),
			},
		})
		return
	}

	// Validation des limites selon spécifications
	if query.Limit > 100 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Limite maximale de 100 enregistrements par page",
			"details": map[string]interface{}{
				"code":      "LIMIT_EXCEEDED",
				"max_limit": 100,
				"provided":  query.Limit,
			},
		})
		return
	}

	result, err := c.service.ListUsers(ctx.Request.Context(), query, establishmentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la récupération des utilisateurs",
			"details": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (c *ComptesController) GetUserDetails(ctx *gin.Context) {
	// Récupération de l'ID utilisateur depuis l'URL
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "ID utilisateur requis",
		})
		return
	}

	// Validation UUID
	if err := c.validator.Var(userID, "uuid"); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Format ID utilisateur invalide",
			"details": map[string]interface{}{
				"code": "INVALID_USER_ID_FORMAT",
				"message": "L'ID utilisateur doit être un UUID valide",
			},
		})
		return
	}

	establishmentCode := ctx.GetHeader("X-Establishment-Code")
	if establishmentCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Header X-Establishment-Code requis",
		})
		return
	}

	establishmentID := ctx.GetString("establishment_id")
	if establishmentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Etablissement non identifié",
		})
		return
	}

	result, err := c.service.GetUserDetails(ctx.Request.Context(), userID, establishmentID)
	if err != nil {
		if strings.Contains(err.Error(), "utilisateur non trouvé") {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Utilisateur non trouvé",
				"details": map[string]interface{}{
					"code": "USER_NOT_FOUND",
					"user_id": userID,
				},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la récupération des détails utilisateur",
			"details": map[string]interface{}{
				"code": "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": result,
	})
}

func (c *ComptesController) ModifyUserPermissions(ctx *gin.Context) {
	// Récupération de l'ID utilisateur depuis l'URL
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "ID utilisateur requis",
		})
		return
	}

	// Validation UUID
	if err := c.validator.Var(userID, "uuid"); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Format ID utilisateur invalide",
			"details": map[string]interface{}{
				"code": "INVALID_USER_ID_FORMAT",
				"message": "L'ID utilisateur doit être un UUID valide",
			},
		})
		return
	}

	establishmentCode := ctx.GetHeader("X-Establishment-Code")
	if establishmentCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Header X-Establishment-Code requis",
		})
		return
	}

	establishmentID := ctx.GetString("establishment_id")
	if establishmentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Etablissement non identifié",
		})
		return
	}

	modifiedByUserID := ctx.GetString("user_id")
	if modifiedByUserID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Utilisateur modificateur non identifié",
		})
		return
	}

	var req dto.ModifyPermissionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Données invalides",
			"details": map[string]interface{}{
				"code": "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Validation des données avec validator
	if err := c.validator.Struct(req); err != nil {
		validationError := &dto.ValidationError{
			Code:   "VALIDATION_ERROR",
			Champs: make(map[string]string),
		}

		for _, fieldErr := range err.(validator.ValidationErrors) {
			fieldName := c.getPermissionFieldName(fieldErr.Field())
			message := c.getValidationMessage(fieldErr)
			validationError.Champs[fieldName] = message
		}

		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Erreur de validation",
			"details": validationError,
		})
		return
	}

	// Validation métier : au moins une modification demandée
	if c.isEmptyPermissionRequest(req) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Aucune modification de permissions spécifiée",
			"details": map[string]interface{}{
				"code": "EMPTY_PERMISSION_REQUEST",
				"message": "Au moins une modification (profils, modules_complets, modules_partiels) doit être spécifiée",
			},
		})
		return
	}

	result, err := c.service.ModifyUserPermissions(ctx.Request.Context(), userID, establishmentID, modifiedByUserID, req)
	if err != nil {
		if strings.Contains(err.Error(), "utilisateur non trouvé") {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Utilisateur non trouvé",
				"details": map[string]interface{}{
					"code": "USER_NOT_FOUND",
					"user_id": userID,
				},
			})
			return
		}

		if strings.Contains(err.Error(), "profil") && strings.Contains(err.Error(), "non trouvé") {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Profil non trouvé",
				"details": map[string]interface{}{
					"code": "PROFILE_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}

		if strings.Contains(err.Error(), "module") && strings.Contains(err.Error(), "non trouvé") {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Module non trouvé",
				"details": map[string]interface{}{
					"code": "MODULE_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}

		if strings.Contains(err.Error(), "rubrique") && strings.Contains(err.Error(), "non trouvée") {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Rubrique non trouvée",
				"details": map[string]interface{}{
					"code": "RUBRIQUE_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Erreur lors de la modification des permissions",
			"details": map[string]interface{}{
				"code": "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": result,
	})
}

func (c *ComptesController) isEmptyPermissionRequest(req dto.ModifyPermissionsRequest) bool {
	// Vérifier si toutes les sections sont vides ou nulles
	if req.Profils != nil && (len(req.Profils.Ajouter) > 0 || len(req.Profils.Retirer) > 0) {
		return false
	}

	if req.ModulesComplets != nil && (len(req.ModulesComplets.Ajouter) > 0 || len(req.ModulesComplets.Retirer) > 0) {
		return false
	}

	if req.ModulesPartiels != nil && (len(req.ModulesPartiels.Ajouter) > 0 || len(req.ModulesPartiels.Modifier) > 0 || len(req.ModulesPartiels.Retirer) > 0) {
		return false
	}

	return true
}

func (c *ComptesController) getPermissionFieldName(fieldName string) string {
	permissionMapping := map[string]string{
		"Profils.Ajouter":              "profils.ajouter",
		"Profils.Retirer":              "profils.retirer",
		"ModulesComplets.Ajouter":      "modules_complets.ajouter",
		"ModulesComplets.Retirer":      "modules_complets.retirer",
		"ModulesPartiels.Ajouter":      "modules_partiels.ajouter",
		"ModulesPartiels.Modifier":     "modules_partiels.modifier",
		"ModulesPartiels.Retirer":      "modules_partiels.retirer",
	}

	if jsonName, exists := permissionMapping[fieldName]; exists {
		return jsonName
	}

	// Fallback pour les champs standards
	return c.getJSONFieldName(fieldName)
}
