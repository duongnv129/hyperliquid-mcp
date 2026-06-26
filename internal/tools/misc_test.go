// internal/tools/misc_test.go
package tools

import (
	"context"
	"testing"
)

type fakeMiscClient struct {
	now int64
}

func (f *fakeMiscClient) ServerTimeMillis() int64 { return f.now }

func TestGetServerTimeHandler(t *testing.T) {
	client := &fakeMiscClient{now: 1700000000000}

	_, out, err := getServerTimeHandler(client)(context.Background(), nil, GetServerTimeArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.ServerTime != 1700000000000 {
		t.Fatalf("getServerTime output = %+v", out)
	}
}
