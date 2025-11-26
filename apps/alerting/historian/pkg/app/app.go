package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grafana/grafana-app-sdk/app"
	"github.com/grafana/grafana-app-sdk/simple"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/grafana/apps/alerting/historian/pkg/apis/alertinghistorian/v0alpha1"
	"github.com/grafana/grafana/apps/alerting/historian/pkg/app/config"
)

type handlerInvoker struct {
	handlers config.Handlers
}

func New(cfg app.Config) (app.App, error) {
	runtimeConfig := cfg.SpecificConfig.(config.RuntimeConfig)

	invoker := handlerInvoker{
		handlers: runtimeConfig.Handlers,
	}

	simpleConfig := simple.AppConfig{
		Name:       "alerting.historian",
		KubeConfig: cfg.KubeConfig,
		VersionedCustomRoutes: map[string]simple.AppVersionRouteHandlers{
			"v0alpha1": {
				{
					Namespaced: true,
					Path:       "/alertstate/query",
					Method:     "POST",
				}: invoker.InvokeGetAlertStateHistoryHandler,
			},
		},
		// TODO: Remove when SDK is fixed.
		ManagedKinds: []simple.AppManagedKind{
			{
				Kind: v0alpha1.DummyKind(),
			},
		},
	}

	a, err := simple.NewApp(simpleConfig)
	if err != nil {
		return nil, err
	}

	err = a.ValidateManifest(cfg.ManifestData)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// InvokeAlertStateQueryHandler handles requests for the GET /something resource route
func (h handlerInvoker) InvokeGetAlertStateHistoryHandler(ctx context.Context, writer app.CustomRouteResponseWriter, request *app.CustomRouteRequest) error {
	var reqBody v0alpha1.CreateAlertstatequeryRequestBody
	err := json.NewDecoder(request.Body).Decode(&reqBody)
	if err != nil {
		return &apierrors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}}
	}

	respBody, err := h.handlers.AlertStateQueryHandler(ctx, reqBody)
	if err != nil {
		return err
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	return json.NewEncoder(writer).Encode(respBody)
}
