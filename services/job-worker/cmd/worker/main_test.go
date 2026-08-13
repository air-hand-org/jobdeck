package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTLSClientRequiresAbsoluteHTTPSURL(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{
		"localhost:9090",
		"http://localhost:9090",
		"https:///missing-host",
	} {
		t.Run(endpoint, func(t *testing.T) {
			conn, err := newTLSClient(endpoint)
			require.Error(t, err)
			require.Nil(t, conn)
		})
	}
}

func TestNewTLSClientAcceptsHTTPSURL(t *testing.T) {
	t.Parallel()

	conn, err := newTLSClient("https://localhost:9090")
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NoError(t, conn.Close())
}
