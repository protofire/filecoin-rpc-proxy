package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqual(t *testing.T) {
	var i interface{} = float64(1)
	var j interface{} = int64(1)
	require.True(t, Equal(i, j))
	i = float64(1)
	j = "2"
	require.False(t, Equal(i, j))
	i = 1.0
	j = 1
	require.True(t, Equal(i, j))
	i = int32(1)
	j = float64(1)
	require.True(t, Equal(i, j))
}
