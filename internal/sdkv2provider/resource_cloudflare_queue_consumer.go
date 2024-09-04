package sdkv2provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/cloudflare/terraform-provider-cloudflare/internal/consts"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func resourceCloudflareQueueConsumer() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceCloudflareQueueConsumerSchema(),
		CreateContext: resourceCloudflareQueueConsumerCreate,
		ReadContext:   resourceCloudflareQueueConsumerRead,
		UpdateContext: resourceCloudflareQueueConsumerUpdate,
		DeleteContext: resourceCloudflareQueueConsumerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceCloudflareQueueConsumerImport,
		},
		Description: "Provides the ability to manage Cloudflare Workers Queue Consumer features.",
	}
}

func resourceCloudflareQueueConsumerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	queueID := d.Get("queue_id").(string)

	req := cloudflare.CreateQueueConsumerParams{
		QueueName: queueID,
		Consumer: cloudflare.QueueConsumer{
			DeadLetterQueue: d.Get("dead_letter_queue").(string),
		},
	}
	if _, ok := d.GetOk("worker"); ok {
		req.Consumer.Type = "worker"
		req.Consumer.ScriptName = d.Get("worker.0.script_name").(string)
		req.Consumer.Environment = d.Get("worker.0.environment").(string)
		req.Consumer.Settings = cloudflare.QueueConsumerSettings{
			BatchSize:   d.Get("worker.0.settings.0.batch_size").(int),
			MaxRetires:  d.Get("worker.0.settings.0.max_retries").(int),
			MaxWaitTime: d.Get("worker.0.settings.0.max_wait_time_ms").(int),
			RetryDelay:  d.Get("worker.0.settings.0.retry_delay").(int),
		}
	} else if _, ok = d.GetOk("http_pull"); ok {
		req.Consumer.Type = "http_pull"
		req.Consumer.Settings = cloudflare.QueueConsumerSettings{
			BatchSize:         d.Get("http_pull.0.settings.0.batch_size").(int),
			MaxRetires:        d.Get("http_pull.0.settings.0.max_retries").(int),
			RetryDelay:        d.Get("http_pull.0.settings.0.retry_delay").(int),
			VisibilityTimeout: d.Get("http_pull.0.settings.0.visibility_timeout_ms").(int),
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating Cloudflare Workers Queue Consumer from struct: %+v", req))

	r, err := client.CreateQueueConsumer(ctx, cloudflare.AccountIdentifier(accountID), req)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error creating workers queue consumer"))
	}

	d.SetId(r.Name)

	tflog.Info(ctx, fmt.Sprintf("Cloudflare Workers Queue Consumer '%s' created for queue with ID: %s", r.Name, queueID))

	return resourceCloudflareQueueConsumerRead(ctx, d, meta)
}

func resourceCloudflareQueueConsumerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	queueID := d.Get("queue_id").(string)

	resp, _, err := client.ListQueueConsumers(ctx, cloudflare.AccountIdentifier(accountID), cloudflare.ListQueueConsumersParams{
		QueueName: queueID,
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error reading queue consumers"))
	}

	if len(resp) == 0 {
		d.SetId("")
		return nil
	}

	var desiredConsumer cloudflare.QueueConsumer
	for _, consumer := range resp {
		if consumer.Name == d.Id() {
			desiredConsumer = consumer
			break
		}
	}
	if desiredConsumer.Name == "" {
		tflog.Debug(ctx, fmt.Sprintf("Cloudflare Workers Queue Consumer with ID %s not found", d.Id()))
		d.SetId("")
		return nil
	}

	if err := d.Set("dead_letter_queue", desiredConsumer.DeadLetterQueue); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("type", desiredConsumer.Type); err != nil {
		return diag.FromErr(err)
	}

	if desiredConsumer.Type == "worker" {
		tflog.Debug(ctx, fmt.Sprintf("Setting worker consumer settings for Cloudflare Workers Queue Consumer with ID %s", d.Id()))
		settings := map[string]interface{}{
			"batch_size":       desiredConsumer.Settings.BatchSize,
			"max_retries":      desiredConsumer.Settings.MaxRetires,
			"max_wait_time_ms": desiredConsumer.Settings.MaxWaitTime,
			"retry_delay":      desiredConsumer.Settings.RetryDelay,
		}
		if err := d.Set("worker", []interface{}{
			map[string]interface{}{
				"script_name": desiredConsumer.Service,
				"environment": desiredConsumer.Environment,
				"settings":    []interface{}{settings},
			},
		}); err != nil {
			return diag.FromErr(err)
		}
	} else if desiredConsumer.Type == "http_pull" {
		tflog.Debug(ctx, fmt.Sprintf("Setting http_pull consumer settings for Cloudflare Workers Queue Consumer with ID %s", d.Id()))
		settings := map[string]interface{}{
			"batch_size":            desiredConsumer.Settings.BatchSize,
			"max_retries":           desiredConsumer.Settings.MaxRetires,
			"retry_delay":           desiredConsumer.Settings.RetryDelay,
			"visibility_timeout_ms": desiredConsumer.Settings.VisibilityTimeout,
		}
		if err := d.Set("http_pull", []interface{}{
			map[string]interface{}{
				"settings": []interface{}{settings},
			},
		}); err != nil {
			return diag.FromErr(err)
		}
	} else {
		tflog.Error(ctx, fmt.Sprintf("Consumer type %qZ not supported", desiredConsumer.Type))
		d.SetId("")
		return nil
	}

	return nil
}

func resourceCloudflareQueueConsumerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	queueID := d.Get("queue_id").(string)

	tflog.Info(ctx, fmt.Sprintf("Updating Cloudflare Workers Queue Consumer with id: %+v", d.Id()))
	updatedConsumer := cloudflare.QueueConsumer{
		Name:            d.Id(),
		DeadLetterQueue: d.Get("dead_letter_queue").(string),
	}

	if _, ok := d.GetOk("worker"); ok {
		updatedConsumer.Type = "worker"
		updatedConsumer.ScriptName = d.Get("worker.0.script_name").(string)
		updatedConsumer.Environment = d.Get("worker.0.environment").(string)
		updatedConsumer.Settings = cloudflare.QueueConsumerSettings{
			BatchSize:   d.Get("worker.0.settings.0.batch_size").(int),
			MaxRetires:  d.Get("worker.0.settings.0.max_retries").(int),
			MaxWaitTime: d.Get("worker.0.settings.0.max_wait_time_ms").(int),
			RetryDelay:  d.Get("worker.0.settings.0.retry_delay").(int),
		}
	} else if _, ok = d.GetOk("http_pull"); ok {
		updatedConsumer.Type = "http_pull"
		updatedConsumer.Settings = cloudflare.QueueConsumerSettings{
			BatchSize:         d.Get("http_pull.0.settings.0.batch_size").(int),
			MaxRetires:        d.Get("http_pull.0.settings.0.max_retries").(int),
			RetryDelay:        d.Get("http_pull.0.settings.0.retry_delay").(int),
			VisibilityTimeout: d.Get("http_pull.0.settings.0.visibility_timeout_ms").(int),
		}
	}

	_, err := client.UpdateQueueConsumer(ctx, cloudflare.AccountIdentifier(accountID), cloudflare.UpdateQueueConsumerParams{
		QueueName: queueID,
		Consumer:  updatedConsumer,
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error updating workers queue consumer"))
	}

	tflog.Debug(ctx, fmt.Sprintf("Updated Cloudflare Workers Queue Consumer with id: %+v", d.Id()))

	return resourceCloudflareQueueConsumerRead(ctx, d, meta)
}

func resourceCloudflareQueueConsumerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	accountID := d.Get(consts.AccountIDSchemaKey).(string)
	queueID := d.Get("queue_id").(string)

	tflog.Info(ctx, fmt.Sprintf("Deleting Cloudflare Workers Queue '%s' Consumer with id: %+v", queueID, d.Id()))

	err := client.DeleteQueueConsumer(ctx, cloudflare.AccountIdentifier(accountID), cloudflare.DeleteQueueConsumerParams{
		QueueName:    queueID,
		ConsumerName: d.Id(),
	})
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error deleting workers queue consumer"))
	}

	d.SetId("")
	return nil
}

func resourceCloudflareQueueConsumerImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	attributes := strings.SplitN(d.Id(), "/", 3)
	if len(attributes) != 3 {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"accountID/queueID/consumerID\"", d.Id())
	}

	accountID, queueID, consumerID := attributes[0], attributes[1], attributes[2]
	tflog.Info(ctx, fmt.Sprintf("Importing Cloudflare Queue Consumer id %s for queue %s account %s", consumerID, queueID, accountID))

	d.SetId(consumerID)
	if err := d.Set(consts.AccountIDSchemaKey, accountID); err != nil {
		return nil, err
	}
	if err := d.Set("queue_id", queueID); err != nil {
		return nil, err
	}

	resourceCloudflareQueueConsumerRead(ctx, d, meta)
	return []*schema.ResourceData{d}, nil
}
