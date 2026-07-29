package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type currentUserService struct {
	user.Service
	current user.User
	err     error
}

func (s currentUserService) Current(
	context.Context,
	security.Actor,
) (user.User, error) {
	return s.current, s.err
}

type accessAuthorizer struct {
	err  error
	code permission.Code
}

func (a *accessAuthorizer) Check(
	_ context.Context,
	_ security.Actor,
	code permission.Code,
) error {
	a.code = code
	return a.err
}

func TestAdminSessionRequiresAuthenticationAndPermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      security.Actor
		accessErr  error
		currentErr error
		wantStatus int
	}{
		{
			name:       "guest",
			actor:      security.Guest(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			actor:      security.User(1),
			accessErr:  security.ErrForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "inactive user",
			actor:      security.User(1),
			currentErr: security.ErrUnauthenticated,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "authorized",
			actor:      security.User(1),
			wantStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			authorizer := &accessAuthorizer{err: test.accessErr}
			runtime := &Runtime{
				users: currentUserService{
					current: user.User{
						ID:    1,
						Login: "admin",
						Email: "admin@example.test",
						Name:  "Администратор",
					},
					err: test.currentErr,
				},
				authorization: authorizer,
			}
			handler, err := runtime.SessionHandler()
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(
				http.MethodGet,
				"/api/admin/session",
				nil,
			)
			request = request.WithContext(httptransport.WithActor(
				request.Context(),
				test.actor,
			))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf(
					"status = %d, body = %q",
					response.Code,
					response.Body.String(),
				)
			}
			if test.actor.IsUser() &&
				authorizer.code != AccessPermission {
				t.Fatalf("permission = %q", authorizer.code)
			}
			if response.Code == http.StatusUnauthorized &&
				response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf(
					"WWW-Authenticate = %q",
					response.Header().Get("WWW-Authenticate"),
				)
			}
		})
	}
}

func TestAdminSessionReturnsSafeCurrentUserPayload(t *testing.T) {
	t.Parallel()

	lastName := "Иванов"
	middleName := "Иванович"
	authorizer := &accessAuthorizer{}
	runtime := &Runtime{
		users: currentUserService{current: user.User{
			ID:         42,
			Login:      "admin",
			Email:      "admin@example.test",
			Name:       "Иван",
			LastName:   &lastName,
			MiddleName: &middleName,
		}},
		authorization: authorizer,
	}
	handler, err := runtime.SessionHandler()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/admin/session",
		nil,
	)
	request = request.WithContext(httptransport.WithActor(
		request.Context(),
		security.User(42),
	))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body = %q",
			response.Code,
			response.Body.String(),
		)
	}
	var payload sessionResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.User.ID != 42 ||
		payload.User.Login != "admin" ||
		payload.User.Email != "admin@example.test" ||
		payload.User.DisplayName != "Иванов Иван Иванович" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAdminAuthorizationMapsUnexpectedErrors(t *testing.T) {
	t.Parallel()

	runtime := &Runtime{
		users: currentUserService{},
		authorization: &accessAuthorizer{
			err: errors.New("database unavailable"),
		},
	}
	handler, err := runtime.SessionHandler()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/admin/session",
		nil,
	)
	request = request.WithContext(httptransport.WithActor(
		request.Context(),
		security.User(1),
	))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, body = %q",
			response.Code,
			response.Body.String(),
		)
	}
}
