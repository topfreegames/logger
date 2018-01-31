// +build testelasticsearch

package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

const (
	indexMapping = `{
                    "mappings" : {
											"doc": {
												"properties": {
													"kubernetes": {
														"properties": {
															"labels": {
																"properties": {
																	"app": { "type": "keyword" }
																}
															},
															"pod": {
																"properties": {
																	"name": { "type": "string" }
																}
															},
															"container": {
																"properties": {
																	"name": { "type": "string" }
																}
															}
														}
													},
													"log": { "type": "string" },
													"@timestamp": { "type": "date" }
												}
											}
                    }
                  }`
)

func TestESReadFromNonExistingApp(t *testing.T) {
	ctx := context.Background()
	a, err := NewESStorageAdapter()
	if err != nil {
		t.Error(err)
	}
	aa := a.(*elasticsearchAdapter)
	client := aa.esClient
	otherApp := fmt.Sprintf("%s-%d", app, 2)
	indexName := fmt.Sprintf(aa.indexTemplate, otherApp)
	client.DeleteIndex(indexName).Do(ctx)

	res, err := client.
		CreateIndex(indexName).
		Body(indexMapping).
		Do(ctx)
	if err != nil {
		t.Error(err)
	}
	if !res.Acknowledged {
		t.Error(
			errors.New("CreateIndex was not acknowledged. Check that timeout value is correct"),
		)
	}

	// No logs have been written; there should be no elasticsearch list for app
	messages, err := a.Read(otherApp, 10, "")
	if messages != nil {
		t.Error("Expected no messages, but got some")
	}
	if err == nil || err.Error() != fmt.Sprintf("Could not find logs for '%s'", otherApp) {
		t.Error("Did not receive expected error message")
	}
}

func TestESLogs(t *testing.T) {
	ctx := context.Background()
	a, err := NewESStorageAdapter()
	if err != nil {
		t.Error(err)
	}
	aa := a.(*elasticsearchAdapter)
	// And write a few logs to it, but do NOT fill it up
	indexName := fmt.Sprintf(aa.indexTemplate, app)
	client := aa.esClient

	client.DeleteIndex(indexName).Do(ctx)

	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error(
			errors.New("Index already exists"),
		)
	}

	res, err := client.
		CreateIndex(indexName).
		Body(indexMapping).
		Do(ctx)
	if err != nil {
		t.Error(err)
	}
	if !res.Acknowledged {
		t.Error(
			errors.New("CreateIndex was not acknowledged. Check that timeout value is correct"),
		)
	}

	writtenMessages := make([]string, 5)
	for i := 0; i < 5; i++ {
		timestamp := fmt.Sprintf("2018-01-22T20:21:0%d.000Z", i)
		l := fmt.Sprintf(`{"kubernetes":{"labels":{"app":"%s"},"pod":{"name":"pod-name"}},"@timestamp":"%s","log":"message %d"}`, app, timestamp, i)
		writtenMessages[i] = fmt.Sprintf("%s %s[pod-name]: message %d", timestamp, app, i)

		_, err := client.Index().
			Index(indexName).
			Type("doc").
			BodyString(l).
			Do(ctx)

		if err != nil {
			t.Error(err)
		}
	}

	// Sleep for a bit because the adapter queues logs internally and writes them to ES only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 3)
	// Read more logs than there are
	messages, err := a.Read(app, 8, "")
	if err != nil {
		t.Error(err)
	}

	if len(messages) != 5 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	// Read fewer logs than there are
	messages, err = a.Read(app, 3, "")
	if err != nil {
		t.Error(err)
	}
	// Should get the 3 MOST RECENT logs
	if len(messages) != 3 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	for i := 0; i < 3; i++ {
		expectedMessage := writtenMessages[i+2]
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}
}

func TestESLogsWithProcess(t *testing.T) {
	ctx := context.Background()
	a, err := NewESStorageAdapter()
	if err != nil {
		t.Error(err)
	}
	aa := a.(*elasticsearchAdapter)
	// And write a few logs to it, but do NOT fill it up
	indexName := fmt.Sprintf(aa.indexTemplate, app)
	client := aa.esClient

	client.DeleteIndex(indexName).Do(ctx)

	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error(
			errors.New("Index already exists"),
		)
	}

	res, err := client.
		CreateIndex(indexName).
		Body(indexMapping).
		Do(ctx)
	if err != nil {
		t.Error(err)
	}
	if !res.Acknowledged {
		t.Error(
			errors.New("CreateIndex was not acknowledged. Check that timeout value is correct"),
		)
	}

	writtenMessages := make([]string, 5)
	for i := 0; i < 5; i++ {
		timestamp := fmt.Sprintf("2018-01-22T20:21:0%d.000Z", i)
		l := fmt.Sprintf(`{"kubernetes":{"labels":{"app":"%s"},"pod":{"name":"pod-name"},"container":{"name":"%s-cmd"}},"@timestamp":"%s","log":"message %d"}`, app, app, timestamp, i)
		writtenMessages[i] = fmt.Sprintf("%s %s[pod-name]: message %d", timestamp, app, i)

		_, err := client.Index().
			Index(indexName).
			Type("doc").
			BodyString(l).
			Do(ctx)

		if err != nil {
			t.Error(err)
		}
	}

	// Sleep for a bit because the adapter queues logs internally and writes them to ES only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 3)
	// Read more logs than there are
	messages, err := a.Read(app, 8, "cmd")
	if err != nil {
		t.Error(err)
	}

	if len(messages) != 5 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	// Read fewer logs than there are
	messages, err = a.Read(app, 3, "")
	if err != nil {
		t.Error(err)
	}
	// Should get the 3 MOST RECENT logs
	if len(messages) != 3 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	for i := 0; i < 3; i++ {
		expectedMessage := writtenMessages[i+2]
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}

	messages, err = a.Read(app, 8, "web")
	if err != nil {
		t.Error(err)
	}

	if len(messages) != 0 {
		t.Errorf("only expected 0 log messages, got %d", len(messages))
	}
}
