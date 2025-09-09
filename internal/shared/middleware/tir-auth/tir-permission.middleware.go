package tir_auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionType énumère les permissions TIR disponibles
type PermissionType string

const (
	PermissionGererLicences                PermissionType = "gerer_licences"
	PermissionGererEtablissements         PermissionType = "gerer_etablissements"
	PermissionAccederDonneesEtablissement PermissionType = "acceder_donnees_etablissement"
	PermissionGererAdminsGlobaux          PermissionType = "gerer_admins_globaux"
)

// TIRPermissionMiddleware vérifie les permissions spécifiques pour les admins TIR
// DOIT être utilisé APRÈS TIRSessionMiddleware
func TIRPermissionMiddleware(requiredPermission PermissionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Vérifier que le middleware TIRSession a été exécuté
		adminID := c.GetString("tir_admin_id")
		if adminID == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Session TIR non initialisée",
				"details": map[string]interface{}{
					"required_middleware": "TIRSessionMiddleware must be called first",
				},
			})
			return
		}

		// Vérifier permission TIR spécifique selon le schéma Redis
		hasPermission := false
		var permissionKey string

		switch requiredPermission {
		case PermissionGererLicences:
			hasPermission = c.GetBool("peut_gerer_licences")
			permissionKey = "peut_gerer_licences"

		case PermissionGererEtablissements:
			hasPermission = c.GetBool("peut_gerer_etablissements")
			permissionKey = "peut_gerer_etablissements"

		case PermissionAccederDonneesEtablissement:
			hasPermission = c.GetBool("peut_acceder_donnees_etablissement")
			permissionKey = "peut_acceder_donnees_etablissement"

		case PermissionGererAdminsGlobaux:
			hasPermission = c.GetBool("peut_gerer_admins_globaux")
			permissionKey = "peut_gerer_admins_globaux"

		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Permission TIR inconnue",
				"details": map[string]interface{}{
					"requested_permission": string(requiredPermission),
					"available_permissions": []string{
						string(PermissionGererLicences),
						string(PermissionGererEtablissements),
						string(PermissionAccederDonneesEtablissement),
						string(PermissionGererAdminsGlobaux),
					},
				},
			})
			return
		}

		// Bloquer si permission insuffisante
		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Permission TIR insuffisante",
				"details": map[string]interface{}{
					"required_permission": string(requiredPermission),
					"permission_key": permissionKey,
					"admin_niveau": c.GetString("tir_niveau_admin"),
					"admin_id": adminID,
				},
			})
			return
		}

		// Permission accordée - enrichir contexte avec info permission validée
		c.Set("validated_tir_permission", string(requiredPermission))
		c.Next()
	}
}

// RequireTIRPermissions middleware helper pour exiger plusieurs permissions
func RequireTIRPermissions(permissions ...PermissionType) []gin.HandlerFunc {
	var middlewares []gin.HandlerFunc
	for _, perm := range permissions {
		middlewares = append(middlewares, TIRPermissionMiddleware(perm))
	}
	return middlewares
}

// TIRSuperAdminOnlyMiddleware middleware pour les opérations super admin uniquement
func TIRSuperAdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Vérifier session active
		adminID := c.GetString("tir_admin_id")
		if adminID == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Session TIR non initialisée",
			})
			return
		}

		// Vérifier niveau super admin
		niveauAdmin := c.GetString("tir_niveau_admin")
		if niveauAdmin != "super_admin_tir" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Accès réservé aux super administrateurs TIR",
				"details": map[string]interface{}{
					"required_level": "super_admin_tir",
					"current_level": niveauAdmin,
					"admin_id": adminID,
				},
			})
			return
		}

		c.Set("validated_super_admin", true)
		c.Next()
	}
}