package services

import "fmt"

// PatientRedisKeys contient les helpers type-safe pour les clés Redis du domaine patient
type PatientRedisKeys struct{}

// NewPatientRedisKeys crée une nouvelle instance des helpers Redis
func NewPatientRedisKeys() *PatientRedisKeys {
	return &PatientRedisKeys{}
}

// PatientSequenceKey génère la clé Redis pour les séquences de génération de codes patient
// Format: soins_suite_{etablissement}_patient_sequence:{year}
func (k *PatientRedisKeys) PatientSequenceKey(etablissementCode string, year int) string {
	return fmt.Sprintf("soins_suite_%s_patient_sequence:%d", etablissementCode, year)
}

// PatientSequenceLockKey génère la clé Redis pour les locks de séquence
// Format: soins_suite_{etablissement}_patient_sequence_lock:{year}
func (k *PatientRedisKeys) PatientSequenceLockKey(etablissementCode string, year int) string {
	return fmt.Sprintf("soins_suite_%s_patient_sequence_lock:%d", etablissementCode, year)
}

// PatientCacheKey génère la clé Redis pour le cache des données patient
// Format: soins_suite_{etablissement}_patient_cache:{code_patient}
func (k *PatientRedisKeys) PatientCacheKey(etablissementCode, codePatient string) string {
	return fmt.Sprintf("soins_suite_%s_patient_cache:%s", etablissementCode, codePatient)
}

// PatientSearchCacheKey génère la clé Redis pour le cache des recherches patient
// Format: soins_suite_{etablissement}_patient_search:{hash_criteres}
func (k *PatientRedisKeys) PatientSearchCacheKey(etablissementCode, hashCriteres string) string {
	return fmt.Sprintf("soins_suite_%s_patient_search:%s", etablissementCode, hashCriteres)
}

// PatientDetailCacheKey génère la clé Redis pour le cache détaillé d'un patient (CS-P-003)
// Format: soins_suite_patient_cache:{code_patient} (selon spécifications Redis)
func (k *PatientRedisKeys) PatientDetailCacheKey(codePatient string) string {
	return fmt.Sprintf("soins_suite_patient_cache:%s", codePatient)
}