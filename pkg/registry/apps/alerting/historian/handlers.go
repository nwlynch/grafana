package historian

import (
	"context"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/apps/alerting/historian/pkg/apis/alertinghistorian/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
)

type Historian interface {
	Query(ctx context.Context, query models.HistoryQuery) (*data.Frame, error)
}

type handlers struct {
	historian Historian
}

func (h *handlers) AlertStateQueryHandler(ctx context.Context, req v0alpha1.CreateAlertstatequeryRequestBody) (v0alpha1.CreateAlertstatequery, error) {
	user, err := identity.GetRequester(ctx)
	if err != nil {
		return v0alpha1.CreateAlertstatequery{},
			&apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusUnauthorized,
					Message: "authentication required",
				}}
	}

	query := models.HistoryQuery{
		OrgID:        user.GetOrgID(),
		SignedInUser: user,
	}

	if req.RuleUID != nil {
		query.RuleUID = *req.RuleUID
	}
	if req.DashboardUID != nil {
		query.DashboardUID = *req.DashboardUID
	}
	if req.PanelID != nil {
		query.PanelID = *req.PanelID
	}
	if req.Previous != nil {
		query.Previous = string(*req.Previous)
	}
	if req.Current != nil {
		query.Current = string(*req.Current)
	}
	if req.From != nil {
		query.From = time.Unix(*req.From, 0)
	}
	if req.To != nil {
		query.To = time.Unix(*req.To, 0)
	}
	if req.Limit != nil {
		query.Limit = int(*req.Limit)
	}
	query.Labels = req.Labels

	frame, err := h.historian.Query(ctx, query)
	if err != nil {
		return v0alpha1.CreateAlertstatequery{},
			&apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusInternalServerError,
					Message: err.Error(),
				}}
	}

	result := v0alpha1.CreateAlertstatequery{
		Entries: make([]v0alpha1.AlertStateEntry, 0, frame.Rows()),
	}

	if frame.Rows() <= 0 {
		return result, nil
	}

	_, lineIdx := frame.FieldByName("Line")
	if lineIdx < 0 {
		return v0alpha1.CreateAlertstatequery{},
			&apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusInternalServerError,
					Message: "no Line field found in historian query response",
				}}
	}
	_, timeIdx := frame.FieldByName("Time")
	if timeIdx < 0 {
		return v0alpha1.CreateAlertstatequery{},
			&apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusInternalServerError,
					Message: "no Time field found in historian query response",
				}}
	}

	for row := 0; row < frame.Rows(); row++ {
		timestamp, ok := frame.At(timeIdx, row).(time.Time)
		if !ok {
			return v0alpha1.CreateAlertstatequery{},
				&apierrors.StatusError{
					ErrStatus: metav1.Status{
						Status:  metav1.StatusFailure,
						Code:    http.StatusInternalServerError,
						Message: "log Time field not a time.Time in historian query response",
					}}
		}
		line, ok := frame.At(lineIdx, row).(string)
		if !ok {
			return v0alpha1.CreateAlertstatequery{},
				&apierrors.StatusError{
					ErrStatus: metav1.Status{
						Status:  metav1.StatusFailure,
						Code:    http.StatusInternalServerError,
						Message: "log Line field not a string in historian query response",
					}}
		}
		result.Entries = append(result.Entries, v0alpha1.AlertStateEntry{
			Timestamp: timestamp.UnixNano(),
			Line:      line,
		})
	}

	return result, nil
}
