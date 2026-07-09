package explore

import (
	"fmt"
	"log"

	myMeili "toentoen-backend/pkg/meilisearch"
	"github.com/meilisearch/meilisearch-go"
)

type ExploreRepository interface {
	SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, int64, error)
}

type exploreRepository struct{}

func NewExploreRepository() ExploreRepository {
	return &exploreRepository{}
}

type SearchRequest struct {
	Keyword string
	Lat     float64
	Lng     float64
	Radius  int // in meters
	Sort    string
	Limit   int64
	Offset  int64
}

type MeiliBusinessDocument struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Products      []string    `json:"products"`
	Rating        float64     `json:"rating"`
	ReviewCount   int64       `json:"review_count"`
	PreviewImages []string    `json:"preview_images"`
	IsOpen        bool        `json:"is_open"`
	CloseTime     string      `json:"close_time"`
	Geo           GeoLocation `json:"_geo"`
	Distance      float64     `json:"_geoDistance,omitempty"`
}

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (r *exploreRepository) SearchBusinesses(req SearchRequest) ([]MeiliBusinessDocument, int64, error) {
	index := myMeili.Client.Index("businesses")

	// Prepare Search Request
	searchReq := &meilisearch.SearchRequest{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	// Geo filtering
	if req.Lat != 0 && req.Lng != 0 {
		// Filter by radius (e.g., 10km)
		searchReq.Filter = []string{
			"_geoRadius(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + ", " + formatInt(req.Radius) + ")",
		}

		if req.Sort == "nearest" {
			searchReq.Sort = []string{
				"_geoPoint(" + formatFloat(req.Lat) + ", " + formatFloat(req.Lng) + "):asc",
			}
		}
	}

	if req.Sort == "rating" {
		searchReq.Sort = []string{"rating:desc"}
	}

	res, err := index.Search(req.Keyword, searchReq)
	if err != nil {
		log.Printf("Meilisearch error: %v", err)
		return nil, 0, err
	}

	var docs []MeiliBusinessDocument
	for _, hit := range res.Hits {
		var doc MeiliBusinessDocument
		if err := hit.DecodeInto(&doc); err == nil {
			// Convert meters to km for response
			doc.Distance = doc.Distance / 1000
			docs = append(docs, doc)
		}
	}

	return docs, res.TotalHits, nil
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
