package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionManager struct {
	client *Client
}

func NewCollectionManager(client *Client) *CollectionManager {
	return &CollectionManager{client: client}
}

func (cm *CollectionManager) CreateFormSchemaCollection(ctx context.Context, module string) error {
	collectionName := fmt.Sprintf("forms_%s", module)
	
	// Schéma de validation pour les formulaires dynamiques
	validator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"form_type", "schema", "created_at"},
			"properties": bson.M{
				"form_type": bson.M{
					"bsonType":    "string",
					"description": "Type du formulaire (ex: patient_admission, consultation)",
				},
				"schema": bson.M{
					"bsonType":    "object",
					"description": "Schéma JSON du formulaire",
				},
				"version": bson.M{
					"bsonType":    "string",
					"description": "Version du schéma",
				},
				"created_at": bson.M{
					"bsonType":    "date",
					"description": "Date de création",
				},
				"updated_at": bson.M{
					"bsonType":    "date",
					"description": "Date de modification",
				},
				"active": bson.M{
					"bsonType":    "bool",
					"description": "Schéma actif ou non",
				},
			},
		},
	}

	opts := options.CreateCollection().SetValidator(validator)
	
	if err := cm.client.CreateCollection(ctx, collectionName, opts); err != nil {
		return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
	}

	// Index sur form_type pour optimiser les requêtes
	return cm.client.CreateIndex(ctx, collectionName, 
		bson.D{{Key: "form_type", Value: 1}}, 
		options.Index().SetUnique(false))
}

func (cm *CollectionManager) CreateDataCollection(ctx context.Context, module, formType string) error {
	collectionName := fmt.Sprintf("data_%s_%s", module, formType)
	
	// Schéma flexible pour les données
	validator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"schema_version", "data", "created_at"},
			"properties": bson.M{
				"schema_version": bson.M{
					"bsonType":    "string",
					"description": "Version du schéma utilisé",
				},
				"data": bson.M{
					"bsonType":    "object",
					"description": "Données du formulaire",
				},
				"user_id": bson.M{
					"bsonType":    "string",
					"description": "ID de l'utilisateur ayant créé l'entrée",
				},
				"establishment_id": bson.M{
					"bsonType":    "string",
					"description": "ID de l'établissement",
				},
				"created_at": bson.M{
					"bsonType":    "date",
					"description": "Date de création",
				},
				"updated_at": bson.M{
					"bsonType":    "date",
					"description": "Date de modification",
				},
			},
		},
	}

	opts := options.CreateCollection().SetValidator(validator)
	
	if err := cm.client.CreateCollection(ctx, collectionName, opts); err != nil {
		return fmt.Errorf("failed to create data collection %s: %w", collectionName, err)
	}

	// Index sur establishment_id et created_at
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "establishment_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
	}

	return cm.client.CreateIndexes(ctx, collectionName, indexes)
}

func (cm *CollectionManager) PrepareModuleCollections(ctx context.Context, modules []string) error {
	for _, module := range modules {
		if err := cm.CreateFormSchemaCollection(ctx, module); err != nil {
			return fmt.Errorf("failed to prepare collections for module %s: %w", module, err)
		}
	}
	return nil
}

func (cm *CollectionManager) ListCollections(ctx context.Context) ([]string, error) {
	return cm.client.ListCollectionNames(ctx)
}

func (cm *CollectionManager) CollectionExists(ctx context.Context, name string) (bool, error) {
	collections, err := cm.client.ListCollectionNames(ctx)
	if err != nil {
		return false, err
	}

	for _, coll := range collections {
		if coll == name {
			return true, nil
		}
	}
	return false, nil
}

func (cm *CollectionManager) DropCollection(ctx context.Context, name string) error {
	return cm.client.DropCollection(ctx, name)
}