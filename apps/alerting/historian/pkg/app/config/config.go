package config

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/historian/pkg/apis/alertinghistorian/v0alpha1"
)

type Handlers interface {
	AlertStateQueryHandler(ctx context.Context, req v0alpha1.CreateAlertstatequeryRequestBody) (v0alpha1.CreateAlertstatequery, error)
}

type RuntimeConfig struct {
	Handlers Handlers
}
