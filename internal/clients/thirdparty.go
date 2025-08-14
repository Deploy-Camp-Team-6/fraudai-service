package clients

import (
	"context"
	"errors"
	"net/http"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var (
	vendorRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vendor_requests_total",
		Help: "Total number of requests to the vendor API.",
	})
	vendorErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vendor_errors_total",
		Help: "Total number of errors from the vendor API.",
	})
)

var tracer = otel.Tracer("third-party-client")

type ThirdPartyClient struct {
	client *resty.Client
	cb     *gobreaker.CircuitBreaker
}

// VersionResponse represents the payload returned by the vendor's
// `/v1/version` endpoint.
type VersionResponse struct {
	BuildTime         string           `json:"build_time"`
	GitSHA            string           `json:"git_sha"`
	MlflowTrackingURI string           `json:"mlflow_tracking_uri"`
	LoadedModels      map[string]Model `json:"loaded_models"`
}

// Model describes a single model entry from the vendor.
type Model struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Stage           string   `json:"stage"`
	RunID           string   `json:"run_id"`
	SignatureInputs []string `json:"signature_inputs"`
}

func NewThirdPartyClient(baseURL, token string, logger zerolog.Logger) *ThirdPartyClient {
	client := resty.New().
		SetBaseURL(baseURL).
		SetAuthToken(token).
		SetTimeout(3 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(50 * time.Millisecond).
		SetRetryMaxWaitTime(2 * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				return r.StatusCode() >= http.StatusInternalServerError
			},
		)

	var st gobreaker.Settings
	st.Name = "third-party-api"
	st.MaxRequests = 1
	st.Interval = 10 * time.Second
	st.Timeout = 5 * time.Second
	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures > 5
	}
	st.OnStateChange = func(name string, from, to gobreaker.State) {
		logger.Info().Str("circuit_breaker", name).Str("from", from.String()).Str("to", to.String()).Msg("circuit breaker state changed")
	}

	cb := gobreaker.NewCircuitBreaker(st)

	return &ThirdPartyClient{
		client: client,
		cb:     cb,
	}
}

func (c *ThirdPartyClient) Ping(ctx context.Context) (string, error) {
	ctx, span := tracer.Start(ctx, "ThirdPartyClient.Ping")
	defer span.End()

	body, err := c.cb.Execute(func() (interface{}, error) {
		vendorRequestsTotal.Inc()
		resp, err := c.client.R().SetContext(ctx).Get("/readyz")
		if err != nil {
			vendorErrorsTotal.Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		if resp.StatusCode() != http.StatusOK {
			err := errors.New("vendor API returned non-200 status")
			vendorErrorsTotal.Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode()))
		return resp.String(), nil
	})
	if err != nil {
		return "", err
	}
	return body.(string), nil
}

// Version fetches metadata about the vendor service including loaded models.
func (c *ThirdPartyClient) Version(ctx context.Context) (*VersionResponse, error) {
	ctx, span := tracer.Start(ctx, "ThirdPartyClient.Version")
	defer span.End()

	body, err := c.cb.Execute(func() (interface{}, error) {
		vendorRequestsTotal.Inc()
		result := &VersionResponse{}
		resp, err := c.client.R().
			SetContext(ctx).
			SetResult(result).
			Get("/v1/version")
		if err != nil {
			vendorErrorsTotal.Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		if resp.StatusCode() != http.StatusOK {
			err := errors.New("vendor API returned non-200 status")
			vendorErrorsTotal.Inc()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode()))
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return body.(*VersionResponse), nil
}
