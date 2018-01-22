package storage

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"gopkg.in/olivere/elastic.v5"
)

type elasticsearchAdapter struct {
	started       bool
	esClient      *elastic.Client
	indexTemplate string
}

// NewESStorageAdapter returns a pointer to a new instance of a elasticsearch-based storage.Adapter.
func NewESStorageAdapter() (Adapter, error) {
	cfg, err := parseESConfig(appName)
	if err != nil {
		log.Fatalf("config error: %s: ", err)
	}

	client, err := elastic.NewClient(
		elastic.SetURL(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)),
	)
	if err != nil {
		panic(err)
	}
	res := &elasticsearchAdapter{
		started:       false,
		esClient:      client,
		indexTemplate: cfg.IndexTemplate,
	}
	return res, nil
}

// Start the storage adapter. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *elasticsearchAdapter) Start() {
}

// Write adds a log message to to an app-specific list in redis using ring-buffer-like semantics
func (a *elasticsearchAdapter) Write(app string, messageBody string) error {
	return nil
}

// Read retrieves a specified number of log lines from an app-specific list in redis
func (a *elasticsearchAdapter) Read(app string, lines int) ([]string, error) {
	ctx := context.Background()
	termQuery := elastic.NewTermQuery("kubernetes.labels.app", app)
	searchResult, err := a.esClient.Search().
		Index(fmt.Sprintf(a.indexTemplate, app)).
		Query(termQuery).
		Sort("@timestamp", false).
		Size(lines).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range searchResult.Each(reflect.TypeOf(map[string]interface{}{})) {
		t := item.(map[string]interface{})
		fmt.Printf("here = %#v\n", t)
		fmt.Println(t["log"].(string))
	}
	return nil, nil
}

// Destroy deletes an app-specific list from redis
func (a *elasticsearchAdapter) Destroy(app string) error {
	return nil
}

// Reopen the storage adapter-- in the case of this implementation, a no-op
func (a *elasticsearchAdapter) Reopen() error {
	return nil
}

// Stop the storage adapter. Additional writes may not be performed after stopping.
func (a *elasticsearchAdapter) Stop() {
}
