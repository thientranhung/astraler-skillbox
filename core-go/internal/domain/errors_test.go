package domain

import (
	"strings"
	"testing"
)

func TestAppErrorFormat(t *testing.T) {
	e := NewValidationError("bad path", "path is not absolute")
	if e.Code != CodeValidation {
		t.Fatalf("code: got %q want %q", e.Code, CodeValidation)
	}
	if !strings.Contains(e.Error(), CodeValidation) {
		t.Fatalf("Error() missing code: %s", e.Error())
	}
	if !strings.Contains(e.Error(), "bad path") {
		t.Fatalf("Error() missing user msg: %s", e.Error())
	}
}

func TestRPCCodes(t *testing.T) {
	cases := []struct {
		factory func() *AppError
		want    int
	}{
		{func() *AppError { return NewValidationError("", "") }, 1001},
		{func() *AppError { return NewFilesystemError("", "") }, 1002},
		{func() *AppError { return NewDatabaseError("", "") }, 1004},
		{func() *AppError { return NewConflictError("", "") }, 1005},
		{func() *AppError { return NewUnknownError("", "") }, 1099},
	}
	for _, tc := range cases {
		e := tc.factory()
		if got := e.RPCCode(); got != tc.want {
			t.Errorf("code %q: RPCCode() = %d, want %d", e.Code, got, tc.want)
		}
	}
}

func TestUnknownCodeFallsBack(t *testing.T) {
	e := &AppError{Code: "not_in_map"}
	if e.RPCCode() != 1099 {
		t.Fatalf("unknown code should return 1099, got %d", e.RPCCode())
	}
}
