package historian

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/apps/alerting/historian/pkg/apis/alertinghistorian/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
)

type mockHistorian struct {
	queryFunc func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error)
}

func (m *mockHistorian) Query(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query)
	}
	return nil, errors.New("not implemented")
}

func TestAlertStateQueryHandler(t *testing.T) {
	t.Run("returns entries when query succeeds", func(t *testing.T) {
		now := time.Now()
		testFrame := data.NewFrame("test",
			data.NewField("Time", nil, []time.Time{now, now.Add(time.Second)}),
			data.NewField("Line", nil, []string{"alert fired", "alert resolved"}),
		)

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				assert.Equal(t, int64(123), query.OrgID)
				assert.NotNil(t, query.SignedInUser)
				return testFrame, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			OrgID: 123,
		})

		result, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.NoError(t, err)
		assert.Len(t, result.Entries, 2)
		assert.Equal(t, now.UnixNano(), result.Entries[0].Timestamp)
		assert.Equal(t, "alert fired", result.Entries[0].Line)
		assert.Equal(t, now.Add(time.Second).UnixNano(), result.Entries[1].Timestamp)
		assert.Equal(t, "alert resolved", result.Entries[1].Line)
	})

	t.Run("returns empty result when frame has no rows", func(t *testing.T) {
		emptyFrame := data.NewFrame("empty")

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return emptyFrame, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		result, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.NoError(t, err)
		assert.Empty(t, result.Entries)
	})

	t.Run("passes all request parameters to historian query", func(t *testing.T) {
		ruleUID := "rule-123"
		dashUID := "dash-456"
		panelID := int64(7)
		fromTime := int64(1000)
		toTime := int64(2000)
		limit := int64(50)
		previous := v0alpha1.CreateAlertstatequeryRequestStatePending
		current := v0alpha1.CreateAlertstatequeryRequestStateAlerting

		testFrame := data.NewFrame("test",
			data.NewField("Time", nil, []time.Time{time.Now()}),
			data.NewField("Line", nil, []string{"test"}),
		)

		var capturedQuery models.HistoryQuery
		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				capturedQuery = query
				return testFrame, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 99})

		req := v0alpha1.CreateAlertstatequeryRequestBody{
			RuleUID:      &ruleUID,
			DashboardUID: &dashUID,
			PanelID:      &panelID,
			From:         &fromTime,
			To:           &toTime,
			Limit:        &limit,
			Previous:     &previous,
			Current:      &current,
			Labels:       map[string]string{"env": "prod"},
		}

		_, err := h.AlertStateQueryHandler(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, ruleUID, capturedQuery.RuleUID)
		assert.Equal(t, dashUID, capturedQuery.DashboardUID)
		assert.Equal(t, panelID, capturedQuery.PanelID)
		assert.Equal(t, time.Unix(fromTime, 0), capturedQuery.From)
		assert.Equal(t, time.Unix(toTime, 0), capturedQuery.To)
		assert.Equal(t, int(limit), capturedQuery.Limit)
		assert.Equal(t, "pending", capturedQuery.Previous)
		assert.Equal(t, "alerting", capturedQuery.Current)
		assert.Equal(t, map[string]string{"env": "prod"}, capturedQuery.Labels)
	})

	t.Run("returns unauthorized when no user in context", func(t *testing.T) {
		h := &handlers{historian: &mockHistorian{}}
		ctx := context.Background()

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusUnauthorized), statusErr.ErrStatus.Code)
		assert.Equal(t, "authentication required", statusErr.ErrStatus.Message)
	})

	t.Run("returns internal error when historian query fails", func(t *testing.T) {
		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return nil, errors.New("database connection failed")
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
		assert.Contains(t, statusErr.ErrStatus.Message, "database connection failed")
	})

	t.Run("returns error when Line field is missing", func(t *testing.T) {
		frameWithoutLine := data.NewFrame("test",
			data.NewField("Time", nil, []time.Time{time.Now()}),
		)

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return frameWithoutLine, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
		assert.Contains(t, statusErr.ErrStatus.Message, "no Line field found")
	})

	t.Run("returns error when Time field is missing", func(t *testing.T) {
		frameWithoutTime := data.NewFrame("test",
			data.NewField("Line", nil, []string{"test"}),
		)

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return frameWithoutTime, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
		assert.Contains(t, statusErr.ErrStatus.Message, "no Time field found")
	})

	t.Run("returns error when Time field has wrong type", func(t *testing.T) {
		frameWithWrongTimeType := data.NewFrame("test",
			data.NewField("Time", nil, []string{"not a time"}),
			data.NewField("Line", nil, []string{"test"}),
		)

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return frameWithWrongTimeType, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
		assert.Contains(t, statusErr.ErrStatus.Message, "Time field not a time.Time")
	})

	t.Run("returns error when Line field has wrong type", func(t *testing.T) {
		frameWithWrongLineType := data.NewFrame("test",
			data.NewField("Time", nil, []time.Time{time.Now()}),
			data.NewField("Line", nil, []float64{123.45}),
		)

		mock := &mockHistorian{
			queryFunc: func(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
				return frameWithWrongLineType, nil
			},
		}

		h := &handlers{historian: mock}
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 1})

		_, err := h.AlertStateQueryHandler(ctx, v0alpha1.CreateAlertstatequeryRequestBody{})

		require.Error(t, err)
		statusErr, ok := err.(*apierrors.StatusError)
		require.True(t, ok)
		assert.Equal(t, int32(http.StatusInternalServerError), statusErr.ErrStatus.Code)
		assert.Contains(t, statusErr.ErrStatus.Message, "Line field not a string")
	})
}
