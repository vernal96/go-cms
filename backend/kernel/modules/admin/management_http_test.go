package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

func TestManagementHTTPListSiteOptionsAndErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		url             string
		denied          map[permission.Code]error
		repositoryError error
		wantStatus      int
		wantCode        string
	}{
		{name: "success", url: "/sites/options?page=1&per_page=10", wantStatus: http.StatusOK},
		{name: "bad pagination", url: "/sites/options?page=zero", wantStatus: http.StatusBadRequest, wantCode: "bad_request"},
		{name: "forbidden", url: "/sites/options", denied: map[permission.Code]error{SiteReadPermission: security.ErrForbidden}, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "database failure", url: "/sites/options", repositoryError: errors.New("pq: secret database detail"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &managementSiteRepository{page: site.Page{
				Items: []site.Site{{ID: 1, Domain: "example.com", ProfileCode: "dev", Locale: "ru-RU"}},
				Total: 1,
			}, err: test.repositoryError}
			management := &Management{
				repository: repository,
				authorizer: managementAuthorizer{denied: test.denied},
				policy:     scopedPolicy{scope: SiteAccessScope{All: true}},
			}
			router := chi.NewRouter()
			registerManagementRoutes(router, management)
			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			request = request.WithContext(httptransport.WithActor(request.Context(), security.User(1)))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "secret database detail") {
				t.Fatalf("database error leaked: %s", response.Body.String())
			}
			if test.wantCode != "" {
				var envelope struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
					t.Fatal(err)
				}
				if envelope.Error.Code != test.wantCode {
					t.Fatalf("error code = %q", envelope.Error.Code)
				}
			}
		})
	}
}
