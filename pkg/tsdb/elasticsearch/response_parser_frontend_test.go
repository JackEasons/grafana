package elasticsearch

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func requireTimeValue(t *testing.T, expected int64, frame *data.Frame, index int) {

	getField := func() *data.Field {
		for _, field := range frame.Fields {
			if field.Type() == data.FieldTypeTime {
				return field
			}
		}
		return nil
	}

	field := getField()
	require.NotNil(t, field, "missing time-field")

	require.Equal(t, time.UnixMilli(expected).UTC(), field.At(index), fmt.Sprintf("wrong time at index %v", index))
}

func requireNumberValue(t *testing.T, expected float64, frame *data.Frame, index int) {

	getField := func() *data.Field {
		for _, field := range frame.Fields {
			if field.Type() == data.FieldTypeNullableFloat64 {
				return field
			}
		}
		return nil
	}

	field := getField()
	require.NotNil(t, field, "missing number-field")

	v := field.At(index).(*float64)

	require.Equal(t, expected, *v, fmt.Sprintf("wrong number at index %v", index))

}

func tRun(text string, callback func(t *testing.T)) {
	// nothing
}

func TestRefIdMatching(t *testing.T) {
	require.NoError(t, nil)
	query := []byte(`
			[
				{
					"timeField": "t",
					"refId": "COUNT_GROUPBY_DATE_HISTOGRAM",
					"metrics": [{ "type": "count", "id": "c_1" }],
					"bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "c_2" }]
				},
				{
					"timeField": "t",
					"refId": "COUNT_GROUPBY_HISTOGRAM",
					"metrics": [{ "type": "count", "id": "h_3" }],
					"bucketAggs": [{ "type": "histogram", "field": "bytes", "id": "h_4" }]
				},
				{
					"timeField": "t",
					"refId": "RAW_DOC",
					"metrics": [{ "type": "raw_document", "id": "r_5" }],
					"bucketAggs": []
				},
				{
					"timeField": "t",
					"refId": "PERCENTILE",
					"metrics": [
					{
						"type": "percentiles",
						"settings": { "percents": ["75", "90"] },
						"id": "p_1"
					}
					],
					"bucketAggs": [{ "type": "date_histogram", "field": "@timestamp", "id": "p_3" }]
				},
				{
					"timeField": "t",
					"refId": "EXTENDEDSTATS",
					"metrics": [
					{
						"type": "extended_stats",
						"meta": { "max": true, "std_deviation_bounds_upper": true },
						"id": "e_1"
					}
					],
					"bucketAggs": [
					{ "type": "terms", "field": "host", "id": "e_3" },
					{ "type": "date_histogram", "id": "e_4" }
					]
				},
				{
					"timeField": "t",
					"refId": "D",
					"metrics": [{ "type": "raw_data", "id": "6" }],
					"bucketAggs": []
				}
			]
			`)

	response := []byte(`
			{
				"responses": [
				  {
					"aggregations": {
					  "c_2": {
						"buckets": [{"doc_count": 10, "key": 1000}]
					  }
					}
				  },
				  {
					"aggregations": {
					  "h_4": {
						"buckets": [{ "doc_count": 1, "key": 1000 }]
					  }
					}
				  },
				  {
					"hits": {
					  "total": 2,
					  "hits": [
						{
						  "_id": "5",
						  "_type": "type",
						  "_index": "index",
						  "_source": { "sourceProp": "asd" },
						  "fields": { "fieldProp": "field" }
						},
						{
						  "_source": { "sourceProp": "asd2" },
						  "fields": { "fieldProp": "field2" }
						}
					  ]
					}
				  },
				  {
					"aggregations": {
					  "p_3": {
						"buckets": [
						  {
							"p_1": { "values": { "75": 3.3, "90": 5.5 } },
							"doc_count": 10,
							"key": 1000
						  },
						  {
							"p_1": { "values": { "75": 2.3, "90": 4.5 } },
							"doc_count": 15,
							"key": 2000
						  }
						]
					  }
					}
				  },
				  {
					"aggregations": {
					  "e_3": {
						"buckets": [
						  {
							"key": "server1",
							"e_4": {
							  "buckets": [
								{
								  "e_1": {
									"max": 10.2,
									"min": 5.5,
									"std_deviation_bounds": { "upper": 3, "lower": -2 }
								  },
								  "doc_count": 10,
								  "key": 1000
								}
							  ]
							}
						  },
						  {
							"key": "server2",
							"e_4": {
							  "buckets": [
								{
								  "e_1": {
									"max": 10.2,
									"min": 5.5,
									"std_deviation_bounds": { "upper": 3, "lower": -2 }
								  },
								  "doc_count": 10,
								  "key": 1000
								}
							  ]
							}
						  }
						]
					  }
					}
				  },
				  {
					"hits": {
					  "total": {
						"relation": "eq",
						"value": 1
					  },
					  "hits": [
						{
						  "_id": "6",
						  "_type": "_doc",
						  "_index": "index",
						  "_source": { "sourceProp": "asd" }
						}
					  ]
					}
				  }
				]
			  }
			`)

	result, err := queryDataTest(query, response)
	require.NoError(t, err)

	verifyFrames := func(name string, expectedLength int) {
		r, found := result.response.Responses[name]
		require.True(t, found, "not found: "+name)
		require.NoError(t, r.Error)
		require.Len(t, r.Frames, expectedLength, "length wrong for "+name)
	}

	verifyFrames("COUNT_GROUPBY_DATE_HISTOGRAM", 1)
	verifyFrames("COUNT_GROUPBY_HISTOGRAM", 1)
	// verifyFrames("RAW_DOC", 1) // FIXME
	verifyFrames("PERCENTILE", 2)
	verifyFrames("EXTENDEDSTATS", 4)
	// verifyFrames("D", 1) // FIXME
}

func TestSimpleQueryReturns1Frame(t *testing.T) {
	query := []byte(`
		[
			{
				"refId": "A",
				"timeField": "t",
				"metrics": [{ "type": "count", "id": "1" }],
				"bucketAggs": [
				{ "type": "date_histogram", "field": "@timestamp", "id": "2" }
				]
			}
			]
		`)

	response := []byte(`
		{
			"responses": [
			  {
				"aggregations": {
				  "2": {
					"buckets": [
					  { "doc_count": 10, "key": 1000 },
					  { "doc_count": 15, "key": 2000 }
					]
				  }
				}
			  }
			]
		  }
		`)

	result, err := queryDataTest(query, response)
	require.NoError(t, err)

	require.Len(t, result.response.Responses, 1)
	frames := result.response.Responses["A"].Frames
	require.Len(t, frames, 1, "frame-count wrong")
	frame := frames[0]
	// require.Equal(t, "Count", frame.Name) // FIXME

	rowLen, err := frame.RowLen()
	require.NoError(t, err)
	require.Equal(t, 2, rowLen)

	requireTimeValue(t, 1000, frame, 0)
	requireNumberValue(t, 10, frame, 0)
}

func TestSimpleQueryCountAndAvg(t *testing.T) {
	query := []byte(`
	[
		{
			"refId": "A",
			"timeField": "t",
			"metrics": [
			{ "type": "count", "id": "1" },
			{ "type": "avg", "field": "value", "id": "2" }
			],
			"bucketAggs": [
			{ "type": "date_histogram", "field": "@timestamp", "id": "3" }
			]
		}
	]
	`)

	response := []byte(`
	{
		"responses": [
		  {
			"aggregations": {
			  "3": {
				"buckets": [
				  { "2": { "value": 88 }, "doc_count": 10, "key": 1000 },
				  { "2": { "value": 99 }, "doc_count": 15, "key": 2000 }
				]
			  }
			}
		  }
		]
	  }
	`)

	result, err := queryDataTest(query, response)
	require.NoError(t, err)

	require.Len(t, result.response.Responses, 1)
	frames := result.response.Responses["A"].Frames
	require.Len(t, frames, 2)

	frame1 := frames[0]
	frame2 := frames[1]

	rowLen1, err := frame1.RowLen()
	require.NoError(t, err)
	require.Equal(t, 2, rowLen1)

	rowLen2, err := frame2.RowLen()
	require.NoError(t, err)
	require.Equal(t, 2, rowLen2)

	requireTimeValue(t, 1000, frame1, 0)
	requireNumberValue(t, 10, frame1, 0)

	// require.Equal(t, "average value", frame2.Name) // FIXME

	requireNumberValue(t, 88, frame2, 0)
	requireNumberValue(t, 99, frame2, 1)
}
