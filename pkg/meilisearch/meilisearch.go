package meilisearch

import (
	"log"

	"toentoen-backend/internal/config"

	"github.com/meilisearch/meilisearch-go"
)

var Client meilisearch.ServiceManager

func ConnectMeilisearch() {
	Client = meilisearch.New(config.Envs.MeiliHost, meilisearch.WithAPIKey(config.Envs.MeiliMasterKey))

	health, err := Client.Health()
	if err != nil {
		log.Printf("⚠️  Meilisearch connection error: %v", err)
	} else {
		log.Printf("🔍 Connected to Meilisearch (%s)", health.Status)
	}
}
