package dnsdelivery

import (
	"errors"
	"fmt"
	"strings"
)

type AssetDelivery struct {
	AssetID   string `json:"asset_id"`
	Domain    string `json:"domain"`
	Origin    string `json:"origin_host"`
	JobStatus string `json:"job_status"`
}

type RecordPlan struct {
	RecordType string `json:"record_type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
}

var ErrProcessing = errors.New("asset processing is not complete")

func PlanCreatorDelivery(asset AssetDelivery) (RecordPlan, error) {
	asset.AssetID = strings.TrimSpace(asset.AssetID)
	asset.Domain = strings.TrimSpace(asset.Domain)
	asset.Origin = strings.TrimSpace(asset.Origin)
	if asset.AssetID == "" || asset.Domain == "" || asset.Origin == "" {
		return RecordPlan{}, errors.New("asset_id, domain, and origin_host are required")
	}
	if asset.JobStatus != "ready" {
		return RecordPlan{}, ErrProcessing
	}

	return RecordPlan{
		RecordType: "CNAME",
		Name:       fmt.Sprintf("%s.%s", asset.AssetID, asset.Domain),
		Content:    asset.Origin,
		TTL:        300,
	}, nil
}
