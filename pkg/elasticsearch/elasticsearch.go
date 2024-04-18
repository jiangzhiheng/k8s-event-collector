package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	v1api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type ESConfig struct {
	Hosts    []string `yaml:"hosts"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

type ESClient struct {
	Client *elasticsearch.Client
	cfg    *ESConfig
}

func NewES(cfg *ESConfig) (*ESClient, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Hosts,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, err
	}

	return &ESClient{
		Client: client,
		cfg:    cfg,
	}, nil
}

func (c *ESClient) checkIndexIsExists(indexName string) bool {
	// 检查索引是否存在
	existsReq := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	existsRes, err := existsReq.Do(context.Background(), c.Client)
	if err != nil {
		klog.Errorf("Error checking index existence: %s", err)
		return false
	}
	defer existsRes.Body.Close()

	if existsRes.IsError() {
		klog.Errorf("Error checking index existence: %s", existsRes.String())
		return false
	}

	if existsRes.StatusCode == 200 {
		return true
	}

	return false
}

func (c *ESClient) CreateIndex(indexName string) {
	// check index
	if c.checkIndexIsExists(indexName) {
		klog.Infof("Index %s already exists, skipping creation", indexName)
		return
	}
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(fmt.Sprintf(CreateIndexEventDocumentBody, IndexILMName, indexName)),
	}
	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		klog.Fatalf("Error creating the index: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		klog.Fatalf("Error creating the index: %s", res.String())
	}
	klog.Infof("index %s Create successfully", indexName)
}

func (c *ESClient) SyncEventItem(event *v1api.Event, indexName string) {
	eventDoc := EventDocument{
		Name:                    event.Name,
		Kind:                    event.Kind,
		Count:                   int64(event.Count),
		InvolvedObjectNamespace: event.InvolvedObject.Namespace,
		InvolvedObjectKind:      event.InvolvedObject.Kind,
		InvolvedObjectName:      event.InvolvedObject.Name,
		Reason:                  event.Reason,
		Message:                 event.Message,
		Type:                    event.Type,
		EventTime:               event.LastTimestamp,
		Action:                  event.Action,
	}
	// 将 EventDocument 转换为 JSON 字节
	eventBytes, err := json.Marshal(eventDoc)
	if err != nil {
		klog.Fatalf("Error marshaling event: %s", err)
	}

	req := esapi.IndexRequest{
		Index: indexName,
		Body:  strings.NewReader(string(eventBytes)),
	}

	// 执行索引请求
	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		klog.Fatalf("Error indexing document: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		klog.Fatalf("Error indexing document: %s", res.String())
	}
}

func (c *ESClient) SearchEventDocuments(namespace, kind, name string) ([]*EventDocument, int64, error) {
	var buf bytes.Buffer
	var r map[string]interface{}
	// 创建查询条件
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"InvolvedObjectNamespace": namespace,
						},
					},
					{
						"match": map[string]interface{}{
							"InvolvedObjectKind": kind,
						},
					},
					{
						"match": map[string]interface{}{
							"InvolvedObjectName": name,
						},
					},
				},
			},
		},
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, 0, fmt.Errorf("error encoding query: %s", err)
	}

	// Perform the search request
	res, err := c.Client.Search(
		c.Client.Search.WithContext(context.Background()),
		c.Client.Search.WithIndex("k8s-event-collector-*"),
		c.Client.Search.WithBody(&buf),
		c.Client.Search.WithTrackTotalHits(true),
		c.Client.Search.WithPretty(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, 0, fmt.Errorf("error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			return nil, 0, fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, 0, fmt.Errorf("error parsing the response body: %s", err)
	}

	klog.Infof(
		"[%s] %d hits; took: %dms",
		res.Status(),
		int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		int(r["took"].(float64)),
	)

	var events []*EventDocument
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		klog.Infof(" * ID=%s, %s", hit.(map[string]interface{})["_id"], hit.(map[string]interface{})["_source"])
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		klog.Infof("source: %v", source)
		event := &EventDocument{
			Type:                    source["Type"].(string),
			Message:                 source["Message"].(string),
			Reason:                  source["Reason"].(string),
			Action:                  source["Action"].(string),
			Name:                    source["Name"].(string),
			Kind:                    source["Kind"].(string),
			RelatedName:             source["RelatedName"].(string),
			RelatedKind:             source["RelatedKind"].(string),
			RelatedNamespace:        source["RelatedNamespace"].(string),
			InvolvedObjectNamespace: source["InvolvedObjectNamespace"].(string),
			InvolvedObjectKind:      source["InvolvedObjectKind"].(string),
			InvolvedObjectName:      source["InvolvedObjectName"].(string),
			Count:                   int64(source["Count"].(float64)),
		}
		eventTime, err := time.Parse(time.RFC3339, source["EventTime"].(string))
		if err != nil {
			klog.Errorf("Error parsing EventTime:", err)

		}
		event.EventTime = v1.NewTime(eventTime)
		events = append(events, event)
	}
	totalHits := int64(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	return events, totalHits, nil
}

func InitIndexTemplate(client *elasticsearch.Client) {
	indexTemplateName := "k8s-event-collector"
	req := esapi.IndicesPutTemplateRequest{
		Name: indexTemplateName,
		Body: strings.NewReader(CreateEventDocumentIndexTemplateBod),
	}

	// Perform the request
	res, err := req.Do(context.Background(), client)
	if err != nil {
		klog.Fatalf("failed to create index template: %v", err)
	}
	defer res.Body.Close()

	// Check the response status
	if res.IsError() {
		klog.Fatalf("failed to create index template: %s", res.String())
	}

	klog.Infof("Index template created: %s", indexTemplateName)
}

func InitIndexILMPolicy(client *elasticsearch.Client) {
	createILMPolicyReq := esapi.ILMPutLifecycleRequest{
		Policy: IndexILMName,
		Body:   strings.NewReader(CreateIndexILMPolicyBody),
	}

	createILMPolicyRes, err := createILMPolicyReq.Do(context.Background(), client)
	if err != nil {
		klog.Fatalf("Error creating index lifecycle policy: %s", err)
	}
	defer createILMPolicyRes.Body.Close()

	if createILMPolicyRes.IsError() {
		klog.Fatalf("Error creating index lifecycle policy: %s", createILMPolicyRes.String())
	}

	klog.Info("Index lifecycle policy created successfully.")
}

/*

func queryData(client *es.Client, namespace, kind, objectName string) {
	// Prepare the query
	var buf bytes.Buffer
	var r map[string]interface{}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"InvolvedObjectNamespace": namespace,
						},
					},
					{
						"match": map[string]interface{}{
							"InvolvedObjectKind": kind,
						},
					},
					{
						"match": map[string]interface{}{
							"InvolvedObjectName": objectName,
						},
					},
				},
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}

	// Perform the search request
	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex("k8s-event-collector-*"),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			log.Fatalf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	// Print the response status, number of results, and request duration.
	log.Printf(
		"[%s] %d hits; took: %dms",
		res.Status(),
		int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		int(r["took"].(float64)),
	)
	// Print the ID and document source for each hit.
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		log.Printf(" * ID=%s, %s", hit.(map[string]interface{})["_id"], hit.(map[string]interface{})["_source"])
	}

	log.Println(strings.Repeat("=", 37))
}

{
  "took": 1,
  "timed_out": false,
  "_shards": {
    "total": 3,
    "successful": 3,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 1,
      "relation": "eq"
    },
    "max_score": 3.9492593,
    "hits": [
      {
        "_index": "k8s-event-collector-2024-04-12",
        "_id": "OVRP0I4BQhn09EwbOZFb",
        "_score": 3.9492593,
        "_source": {
          "Type": "Normal",
          "Message": "Stopping container server",
          "Reason": "Killing",
          "Action": "",
          "Name": "argocd-server-7965b94c48-z99hk.17c56a0cb01da61c",
          "Kind": "",
          "RelatedName": "",
          "RelatedKind": "",
          "RelatedNamespace": "",
          "InvolvedObjectNamespace": "argocd",
          "InvolvedObjectKind": "Pod",
          "InvolvedObjectName": "argocd-server-7965b94c48-z99hk",
          "EventTime": "2024-04-12T03:17:16Z",
          "Count": 1
        }
      }
    ]
  }
}

*/
