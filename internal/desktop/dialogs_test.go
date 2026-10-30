package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestDialogResultPath(t *testing.T) {
	failure := errors.New("dialog failed")
	tests := []struct {
		name     string
		path     string
		err      error
		wantPath string
		wantErr  error
	}{
		{"picked", `C:\certs\a.pfx`, nil, `C:\certs\a.pfx`, nil},
		{"canceled", "", errors.New(dialogCanceledMessage), "", nil},
		{"failed", "", failure, "", failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := dialogResult(tt.path, tt.err)
			if path != tt.wantPath || !errors.Is(err, tt.wantErr) {
				t.Errorf("dialogResult = (%q, %v), want (%q, %v)", path, err, tt.wantPath, tt.wantErr)
			}
		})
	}
}

func TestDialogResultPaths(t *testing.T) {
	failure := errors.New("dialog failed")
	picked := []string{`C:\xml\a.xml`, `C:\xml\b.xml`}
	tests := []struct {
		name      string
		paths     []string
		err       error
		wantPaths []string
		wantErr   error
	}{
		{"picked", picked, nil, picked, nil},
		{"canceled", nil, errors.New(dialogCanceledMessage), nil, nil},
		{"failed", nil, failure, nil, failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := dialogResult(tt.paths, tt.err)
			if !reflect.DeepEqual(paths, tt.wantPaths) || !errors.Is(err, tt.wantErr) {
				t.Errorf("dialogResult = (%q, %v), want (%q, %v)", paths, err, tt.wantPaths, tt.wantErr)
			}
		})
	}
}
