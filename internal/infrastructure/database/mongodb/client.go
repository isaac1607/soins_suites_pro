package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

type MongoConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

func NewClient(config *MongoConfig) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.URI)

	// Configuration du pool de connexions
	clientOptions.SetMaxPoolSize(100)
	clientOptions.SetMinPoolSize(5)
	clientOptions.SetMaxConnIdleTime(30 * time.Minute)
	clientOptions.SetConnectTimeout(10 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)

	// Options optimisées pour performance
	clientOptions.SetReadPreference(readpref.SecondaryPreferred())
	clientOptions.SetRetryWrites(true)
	clientOptions.SetRetryReads(true)

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	client := &Client{
		client:   mongoClient,
		database: mongoClient.Database(config.Database),
	}

	// Test de connexion
	if err := client.Ping(ctx); err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("MongoDB client is nil")
	}

	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (c *Client) Close(ctx context.Context) error {
	if c.client != nil {
		return c.client.Disconnect(ctx)
	}
	return nil
}

func (c *Client) Client() *mongo.Client {
	return c.client
}

func (c *Client) Database() *mongo.Database {
	return c.database
}

func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

func (c *Client) CreateCollection(ctx context.Context, name string, opts ...*options.CreateCollectionOptions) error {
	return c.database.CreateCollection(ctx, name, opts...)
}

func (c *Client) DropCollection(ctx context.Context, name string) error {
	return c.database.Collection(name).Drop(ctx)
}

func (c *Client) ListCollectionNames(ctx context.Context) ([]string, error) {
	return c.database.ListCollectionNames(ctx, nil)
}

func (c *Client) CreateIndex(ctx context.Context, collection string, keys interface{}, opts ...*options.IndexOptions) error {
	coll := c.Collection(collection)
	indexModel := mongo.IndexModel{
		Keys: keys,
	}
	if len(opts) > 0 {
		indexModel.Options = opts[0]
	}

	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (c *Client) CreateIndexes(ctx context.Context, collection string, models []mongo.IndexModel) error {
	coll := c.Collection(collection)
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func (c *Client) DropIndex(ctx context.Context, collection, name string) error {
	coll := c.Collection(collection)
	_, err := coll.Indexes().DropOne(ctx, name)
	return err
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.Ping(ctx); err != nil {
		return err
	}

	// Test d'écriture/lecture simple
	testCollection := c.Collection("_health_check")

	// Test d'insertion
	testDoc := map[string]interface{}{
		"timestamp": time.Now(),
		"test":      "health_check",
	}

	result, err := testCollection.InsertOne(ctx, testDoc)
	if err != nil {
		return fmt.Errorf("health check insert failed: %w", err)
	}

	// Test de lecture
	var readDoc map[string]interface{}
	err = testCollection.FindOne(ctx, map[string]interface{}{"_id": result.InsertedID}).Decode(&readDoc)
	if err != nil {
		return fmt.Errorf("health check read failed: %w", err)
	}

	// Nettoyage
	_, err = testCollection.DeleteOne(ctx, map[string]interface{}{"_id": result.InsertedID})
	if err != nil {
		return fmt.Errorf("health check cleanup failed: %w", err)
	}

	return nil
}

func (c *Client) Stats(ctx context.Context) (map[string]interface{}, error) {
	result := c.database.RunCommand(ctx, map[string]interface{}{"dbStats": 1})
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", result.Err())
	}

	var stats map[string]interface{}
	if err := result.Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return stats, nil
}
