package smartapi

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func Test_handleErrorValue(t *testing.T) {
	type args struct {
		logger     Logger
		errorValue reflect.Value
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name         string
		args         args
		responseCode int
		responseBody string
	}{
		{
			name: "OK",
			args: args{
				logger:     nil,
				errorValue: reflect.ValueOf(Error(404, "not found", "not found")),
			},
			responseCode: 404,
			responseBody: "{\"status\":404,\"reason\":\"not found\"}\n",
		},
		{
			name: "Non Error Passed",
			args: args{
				logger:     nil,
				errorValue: reflect.ValueOf(32),
			},
			responseCode: 500,
			responseBody: "{\"status\":500,\"reason\":\"invalid API construction\"}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handleErrorValue(context.Background(), rr, tt.args.logger, tt.args.errorValue)
			require.Equal(t, tt.responseCode, rr.Code)
			require.Equal(t, tt.responseBody, rr.Body.String())
		})
	}
}
