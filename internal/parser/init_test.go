package parser_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createTestFile(t *testing.T, content string) string {
	fileName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	f, err := os.Create(fileName)
	require.NoError(t, err, "file must be created")

	fmt.Fprint(f, content)

	defer func() {
		err := f.Close()
		require.NoError(t, err, "file must be closed")
	}()

	return fileName
}

func deleteTestFile(t *testing.T, fileName string) {
	err := os.Remove(fileName)
	require.NoError(t, err, "file should be removed")
}
