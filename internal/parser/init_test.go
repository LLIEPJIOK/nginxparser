package parser_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const timeLayout = "02/Jan/2006"

func createTestFiles(t *testing.T, content ...string) string {
	dir, err := os.MkdirTemp("", "test_*")
	require.NoError(t, err, "dir should be created")

	for _, c := range content {
		fileName := fmt.Sprintf("%s/test_%d", dir, time.Now().UnixNano())

		f, err := os.Create(fileName)
		require.NoError(t, err, "file must be created")

		fmt.Fprint(f, c)

		err = f.Close()
		require.NoError(t, err, "file must be closed")

		time.Sleep(time.Nanosecond)
	}

	return fmt.Sprintf("%s/test_*", dir)
}

func deleteTestFiles(t *testing.T, path string) {
	err := os.RemoveAll(path)
	require.NoError(t, err, "path should be removed")
}

func getRoot(path string) string {
	if slashIndex := strings.Index(path, "/"); slashIndex != -1 {
		return path[:slashIndex]
	}

	return path
}

func getTime(t *testing.T, value string) *time.Time {
	tm, err := time.Parse(timeLayout, value)
	require.NoError(t, err, "time must be parsed")

	return &tm
}
