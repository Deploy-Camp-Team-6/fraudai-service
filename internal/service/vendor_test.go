package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jules-labs/go-api-prod-template/internal/clients"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVendorService_ListModels(t *testing.T) {
	response := `{"build_time":"dev","git_sha":"dev","mlflow_tracking_uri":"https://mlflow.fraudai.cloud","loaded_models":{"lightgbm":{"name":"FraudDetector-lightgbm","version":"1","stage":"None","run_id":"e3bc6f35cd3e4a2a977b62e2ffe5e181","signature_inputs":["transaction_id","amount","merchant_type","device_type"]},"xgboost":{"name":"FraudDetector-xgboost","version":"2","stage":"None","run_id":"76811460051f4229809b24d771a2ce2c","signature_inputs":["transaction_id","amount","merchant_type","device_type"]},"logreg":{"name":"FraudDetector-logistic_regression","version":"1","stage":"None","run_id":"121e6c15715c420b8f0b9139d75fd30d","signature_inputs":["transaction_id","amount","merchant_type","device_type"]}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/version" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := clients.NewThirdPartyClient(server.URL, "", zerolog.Nop())
	svc := NewVendorService(client)

	models, err := svc.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 3)
	assert.Equal(t, "lightgbm", models[0].ModelType)
	assert.Equal(t, "FraudDetector-lightgbm", models[0].Name)
}
