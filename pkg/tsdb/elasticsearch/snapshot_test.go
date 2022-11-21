package elasticsearch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/stretchr/testify/require"
)

// these snapshot-tests test the whole request-response flow:
// the inputs:
// - the backend.DataQuery query
// - the elastic-response json
// the snapshot verifies:
// - the elastic-request json
// - the dataframe result

// a regex that matches the request-snapshot-filenames, and extracts the name of the test
var requestRe = regexp.MustCompile(`^(.*)\.request\.line\d+\.json$`)

// the "elastic request" is often in multiple json-snapshot-files,
// so we have to find them on disk, so we have to look at every file in
// the folder.
func findRequestSnapshots(t *testing.T) map[string][]string {
	allTestSnapshotFiles, err := os.ReadDir("testdata")
	require.NoError(t, err)

	snapshots := make(map[string][]string)

	for _, file := range allTestSnapshotFiles {
		fileName := file.Name()
		match := requestRe.FindStringSubmatch(fileName)
		if len(match) == 2 {
			testName := match[1]
			files := append(snapshots[testName], filepath.Join("testdata", fileName))
			snapshots[testName] = files
		}
	}

	return snapshots
}

func TestSnapshots(t *testing.T) {

	tt := []struct {
		name string
		path string
	}{
		{name: "simple metric test", path: "metric_simple"},
		{name: "complex metric test", path: "metric_complex"},
	}

	queryHeader := []byte(`
	{
		"ignore_unavailable": true,
		"index": "testdb-2022.11.14",
		"search_type": "query_then_fetch"
	}
	`)

	requestSnapshots := findRequestSnapshots(t)

	for _, test := range tt {

		t.Run(test.name, func(t *testing.T) {
			goldenFileName := test.path + ".golden"

			responseFileName := filepath.Join("testdata", test.path+".response.json")
			responseBytes, err := os.ReadFile(responseFileName)
			require.NoError(t, err)

			queriesFileName := filepath.Join("testdata", test.path+".queries.json")
			queriesBytes, err := os.ReadFile(queriesFileName)
			require.NoError(t, err)

			var requestLines [][]byte

			for _, fileName := range requestSnapshots[test.path] {
				bytes, err := os.ReadFile(fileName)
				require.NoError(t, err)
				requestLines = append(requestLines, bytes)
			}

			require.True(t, len(requestLines) > 0, "requestLines must not be empty")

			result, err := queryDataTest(queriesBytes, responseBytes)
			require.NoError(t, err)

			reqLines := strings.Split(strings.TrimSpace(string(result.requestBytes)), "\n")
			require.Len(t, reqLines, len(requestLines)*2)

			for i, expectedRequestLine := range requestLines {
				actualRequestHeaderLine := reqLines[2*i]
				actualRequestLine := reqLines[2*i+1]
				require.JSONEq(t, string(queryHeader), string(actualRequestHeaderLine))
				require.JSONEq(t, string(expectedRequestLine), string(actualRequestLine))
			}

			require.NoError(t, err)

			require.Len(t, result.response.Responses, 1)

			queryRes := result.response.Responses["A"]
			require.NotNil(t, queryRes)

			experimental.CheckGoldenJSONResponse(t, "testdata", goldenFileName, &queryRes, false)
		})
	}

}
