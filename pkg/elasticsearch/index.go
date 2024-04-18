package elasticsearch

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type EventDocument struct {
	Type                    string
	Message                 string
	Reason                  string
	Action                  string
	Name                    string
	Kind                    string
	RelatedName             string
	RelatedKind             string
	RelatedNamespace        string
	InvolvedObjectNamespace string
	InvolvedObjectKind      string
	InvolvedObjectName      string
	EventTime               v1.Time
	Count                   int64
}

const IndexILMName = "K8sEventCollectorILM"

const CreateEventDocumentIndexTemplateBod string = `
{
  "index_patterns": ["k8s-event-collector*"],
  "mappings": {
    "properties": {
      "Type": { "type": "keyword" },
      "Message": { "type": "text" },
      "Reason": { "type": "keyword" },
      "Action": { "type": "keyword" },
      "Name": { "type": "keyword" },
      "Kind": { "type": "keyword" },
      "RelatedName": { "type": "keyword" },
      "RelatedKind": { "type": "keyword" },
      "RelatedNamespace": { "type": "keyword" },
      "InvolvedObjectNamespace": { "type": "keyword" },
      "InvolvedObjectKind": { "type": "keyword" },
      "InvolvedObjectName": { "type": "keyword" },
      "EventTime": { "type": "date" },
      "Count": { "type": "long" }
    }
  }
}
`

const CreateIndexEventDocumentBody string = `
{
	"settings": {
		"index.lifecycle.name": "%s",
		"index.lifecycle.rollover_alias": "%s"
	}
}
`

const CreateIndexILMPolicyBody string = `
		{
			"policy": {
				"phases": {
					"hot": {
						"actions": {
							"rollover": {
								"max_age": "1d"
							}
						}
					},
					"delete": {
						"min_age": "3d",
						"actions": {
							"delete": {}
						}
					}
				}
			}
		}
`
