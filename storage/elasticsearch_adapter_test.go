// +build testelasticsearch

package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestESReadFromNonExistingApp(t *testing.T) {
	a, err := NewESStorageAdapter()
	if err != nil {
		t.Error(err)
	}
	// No logs have been written; there should be no elasticsearch list for app
	messages, err := a.Read(app, 10)
	if messages != nil {
		t.Error("Expected no messages, but got some")
	}
	if err == nil || err.Error() != fmt.Sprintf("Could not find logs for '%s'", app) {
		t.Error("Did not receive expected error message")
	}
}

// const (
// 	indexMapping = `{
//                     "mappings" : {
// 											"kubernetes": {
// 												"properties": {
// 													"labels": {
// 														"type": "nested",
// 														"properties": {
// 															"app": { "type": "string" }
// 														}
// 													}
// 												}
// 											}
//                     }
//                   }`
// )

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
															}
														}
													},
													"log": { "type": "string" }
												}
											}
                    }
                  }`
)

func TestESLogs(t *testing.T) {
	ctx := context.Background()
	a, err := NewESStorageAdapter()
	if err != nil {
		t.Error(err)
	}
	aa := a.(*elasticsearchAdapter)
	a.Start()
	defer a.Stop()
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

	for i := 0; i < 5; i++ {
		l := fmt.Sprintf(`{"kubernetes":{"labels":{"app":"%s"}}}`, app)

		res, err := client.Index().
			Index(indexName).
			Type("doc").
			BodyString(l).
			Do(ctx)

		fmt.Printf("res: %#v\n", res)

		if err != nil {
			t.Error(err)
		}
	}

	// Sleep for a bit because the adapter queues logs internally and writes them to ES only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 3)
	// Read more logs than there are
	messages, err := a.Read(app, 1)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%#v\n", messages)

	if len(messages) != 5 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	// Read fewer logs than there are
	messages, err = a.Read(app, 3)
	if err != nil {
		t.Error(err)
	}
	// Should get the 3 MOST RECENT logs
	if len(messages) != 3 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	for i := 0; i < 3; i++ {
		expectedMessage := fmt.Sprintf("message %d", i+2)
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}
	// Overfill the buffer
	for i := 5; i < 11; i++ {
		if err := a.Write(app, fmt.Sprintf("message %d", i)); err != nil {
			t.Error(err)
		}
	}
	// Sleep for a bit because the adapter queues logs internally and writes them to ES only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 2)
	// Read more logs than the buffer can hold
	messages, err = a.Read(app, 20)
	if err != nil {
		t.Error(err)
	}
	// Should only get as many messages as the buffer can hold
	if len(messages) != 10 {
		t.Errorf("only expected 10 log messages, got %d", len(messages))
	}
	// And they should only be the 10 MOST RECENT logs
	for i := 0; i < 10; i++ {
		expectedMessage := fmt.Sprintf("message %d", i+1)
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}
}
