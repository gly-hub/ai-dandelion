package dao

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteAPIUsesClientAndAPIKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPI{}); err != nil {
		t.Fatal(err)
	}
	store := NewExternalAPI(db)
	target := &model.ExternalAPI{UUID: "api-1", ClientKey: "game-server", APIKey: "game.order.get", Name: "Get order", Method: "GET", Path: "/orders/{id}", HeadersJSON: "{}", RequestSchemaJSON: "{}", ResponseSchemaJSON: "{}", Status: "enabled"}
	other := &model.ExternalAPI{UUID: "api-2", ClientKey: "game-server", APIKey: "game.order.list", Name: "List orders", Method: "GET", Path: "/orders", HeadersJSON: "{}", RequestSchemaJSON: "{}", ResponseSchemaJSON: "{}", Status: "enabled"}
	if err := store.CreateAPI(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAPI(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteAPI(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetAPI(context.Background(), target.ClientKey, target.APIKey); err == nil {
		t.Fatal("target API still exists")
	}
	if _, err := store.GetAPI(context.Background(), other.ClientKey, other.APIKey); err != nil {
		t.Fatalf("other API was deleted: %v", err)
	}
}

func TestSoftDeleteAndPurgeClientAssets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPIClient{}, &model.ExternalAPIGroup{}, &model.ExternalAPI{}); err != nil {
		t.Fatal(err)
	}
	store := NewExternalAPI(db)
	client := &model.ExternalAPIClient{UUID: "client-1", ClientKey: "game-server", Name: "Game", BaseURL: "https://game.example.com", DefaultHeadersJSON: "{}", Status: "enabled"}
	group := &model.ExternalAPIGroup{UUID: "group-1", ClientKey: client.ClientKey, Name: "Orders"}
	api := &model.ExternalAPI{UUID: "api-1", ClientKey: client.ClientKey, GroupID: group.UUID, APIKey: "game.orders.get", Name: "Get", Method: "GET", Path: "/orders", HeadersJSON: "{}", RequestSchemaJSON: "{}", ResponseSchemaJSON: "{}", Status: "enabled"}
	if err := store.CreateClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGroup(context.Background(), group); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAPI(context.Background(), api); err != nil {
		t.Fatal(err)
	}
	if err := store.SoftDeleteClientAssets(context.Background(), client.ClientKey, "admin", 100); err != nil {
		t.Fatal(err)
	}
	if rows, err := store.ListClients(context.Background()); err != nil || len(rows) != 0 {
		t.Fatalf("active clients = %#v, %v", rows, err)
	}
	if rows, err := store.ListAPIs(context.Background(), client.ClientKey); err != nil || len(rows) != 0 {
		t.Fatalf("active APIs = %#v, %v", rows, err)
	}
	if rows, err := store.ListDeletedClients(context.Background()); err != nil || len(rows) != 1 {
		t.Fatalf("recycle bin = %#v, %v", rows, err)
	}
	if err := store.PurgeClientAssets(context.Background(), client.ClientKey); err != nil {
		t.Fatal(err)
	}
	var remaining int64
	if err := db.Model(&model.ExternalAPI{}).Where("client_key = ?", client.ClientKey).Count(&remaining).Error; err != nil || remaining != 0 {
		t.Fatalf("remaining APIs = %d, %v", remaining, err)
	}
	if err := db.Model(&model.ExternalAPIClient{}).Where("client_key = ?", client.ClientKey).Count(&remaining).Error; err != nil || remaining != 0 {
		t.Fatalf("remaining clients = %d, %v", remaining, err)
	}
}

func TestUpdateClientUsesClientKeyInsteadOfInsertingByPrimaryKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPIClient{}); err != nil {
		t.Fatal(err)
	}
	store := NewExternalAPI(db)
	client := &model.ExternalAPIClient{UUID: "client-1", ClientKey: "game-server", Name: "Game", BaseURL: "https://game.example.com", DefaultHeadersJSON: "{}", Status: "enabled"}
	if err := store.CreateClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	// Models returned by a legacy query can lack the numeric primary key. The
	// client key remains the stable update identity and must not cause an insert.
	legacyRow := &model.ExternalAPIClient{UUID: "client-1", ClientKey: "game-server", Name: "Game updated", BaseURL: "https://game.example.com", DefaultHeadersJSON: "{}", Status: "enabled", SwaggerImportKeyHash: "hash"}
	if err := store.UpdateClient(context.Background(), legacyRow); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListClients(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Name != "Game updated" || rows[0].SwaggerImportKeyHash != "hash" {
		t.Fatalf("client update = %#v, %v", rows, err)
	}
}
