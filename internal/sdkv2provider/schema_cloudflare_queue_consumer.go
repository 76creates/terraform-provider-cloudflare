package sdkv2provider

import (
	"github.com/cloudflare/terraform-provider-cloudflare/internal/consts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	queueConsumerHttpPullSettingsElem = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"settings": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: "Consumer settings.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"batch_size": {
							Description: "Batch size",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     10,
						},
						"max_retries": {
							Description: "Max retries",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     3,
						},
						"retry_delay": {
							Description: "Retry delay in seconds",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
						},
						"visibility_timeout_ms": {
							Description: "Visibility timeout in milliseconds, defined for HTTP pull consumers",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     30000,
						},
					},
				},
			},
		},
	}

	queueConsumerWorkerSettingsElem = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"script_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the script/service containing the function for consuming the queue.",
			},
			"environment": {
				Description: "Environment name",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "production",
			},
			"settings": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: "Consumer settings.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"batch_size": {
							Description: "Batch size",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     10,
						},
						"max_retries": {
							Description: "Max retries",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     3,
						},
						"max_wait_time_ms": {
							Description: "Max wait time in milliseconds",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     5000,
						},
						"retry_delay": {
							Description: "Retry delay in seconds",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
						},
					},
				},
			},
		},
	}
)

func resourceCloudflareQueueConsumerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"account_id": {
			Description: consts.AccountIDSchemaDescription,
			Type:        schema.TypeString,
			Required:    true,
		},
		"queue_id": {
			Description: "The queue identifier to target for the resource.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"type": {
			Description: "Type of the consumer, can be either `worker` or `http_pull`.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		"dead_letter_queue": {
			Description: "Dead letter queue",
			Type:        schema.TypeString,
			Optional:    true,
		},
		"worker": {
			Type:        schema.TypeList,
			ForceNew:    true,
			Optional:    true,
			MaxItems:    1,
			Description: "Worker settings.",
			Elem:        queueConsumerWorkerSettingsElem,
		},
		"http_pull": {
			Type:         schema.TypeList,
			ForceNew:     true,
			Optional:     true,
			MaxItems:     1,
			Description:  "HTTP pull settings.",
			Elem:         queueConsumerHttpPullSettingsElem,
			ExactlyOneOf: []string{"worker", "http_pull"},
		},
	}
}
