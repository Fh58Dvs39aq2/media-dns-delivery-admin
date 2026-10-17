package dnsdelivery

import (
	"errors"
	"testing"
)

func TestPlanCreatorDelivery(t *testing.T) {
	tests := []struct {
		name    string
		asset   AssetDelivery
		want    RecordPlan
		wantErr error
	}{
		{
			name:  "ready asset gets creator hostname",
			asset: AssetDelivery{AssetID: "clip-42", Domain: "media.example.com", Origin: "edge.example.net", JobStatus: "ready"},
			want:  RecordPlan{RecordType: "CNAME", Name: "clip-42.media.example.com", Content: "edge.example.net", TTL: 300},
		},
		{
			name:    "processing asset is held",
			asset:   AssetDelivery{AssetID: "clip-42", Domain: "media.example.com", Origin: "edge.example.net", JobStatus: "processing"},
			wantErr: ErrProcessing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanCreatorDelivery(tt.asset)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("plan = %#v, want %#v", got, tt.want)
			}
		})
	}
}
