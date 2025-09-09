package services

// ServiceError - Erreur métier commune pour tous les services du core-service establishment
type ServiceError struct {
	Type    string                 `json:"type"`    // "validation", "not_found", "conflict"
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

func (e *ServiceError) Error() string {
	return e.Message
}