package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGameliftFleet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftFleetCreate,
		Read:   resourceAwsGameliftFleetRead,
		Update: resourceAwsGameliftFleetUpdate,
		Delete: resourceAwsGameliftFleetDelete,

		// TODO: Timeout

		Schema: map[string]*schema.Schema{
			"build_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ec2_instance_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ec2_inbound_permissions": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"log_paths": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"metric_groups": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"new_game_session_protection_policy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"peer_vpc_aws_account_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"peer_vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"resource_creation_limit_policy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"routing_strategy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"runtime_configuration": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_launch_parameters": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_launch_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsGameliftFleetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	input := gamelift.CreateFleetInput{
		BuildId:         aws.String(d.Get("build_id").(string)),
		EC2InstanceType: aws.String(d.Get("ec2_instance_type").(string)),
		Name:            aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("ec2_inbound_permissions"); ok {
		input.EC2InboundPermissions = expandGameliftIpPermissions(v.([]interface{}))
	}
	if v, ok := d.GetOk("log_paths"); ok {
		input.LogPaths = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("metric_groups"); ok {
		input.MetricGroups = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("new_game_session_protection_policy"); ok {
		input.NewGameSessionProtectionPolicy = aws.String(v.(string))
	}
	if v, ok := d.GetOk("peer_vpc_aws_account_id"); ok {
		input.PeerVpcAwsAccountId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("peer_vpc_id"); ok {
		input.PeerVpcId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("resource_creation_limit_policy"); ok {
		input.ResourceCreationLimitPolicy = expandGameliftResourceCreationLimitPolicy(v.([]interface{}))
	}
	if v, ok := d.GetOk("runtime_configuration"); ok {
		input.RuntimeConfiguration = expandGameliftRuntimeConfiguration(v.([]interface{}))
	}
	if v, ok := d.GetOk("server_launch_parameters"); ok {
		input.ServerLaunchParameters = aws.String(v.(string))
	}
	if v, ok := d.GetOk("server_launch_path"); ok {
		input.ServerLaunchPath = aws.String(v.(string))
	}

	log.Printf("[INFO] Creating Gamelift Fleet: %s", input)
	out, err := conn.CreateFleet(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.FleetAttributes.FleetId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			gamelift.FleetStatusActivating,
			gamelift.FleetStatusBuilding,
			gamelift.FleetStatusDownloading,
			gamelift.FleetStatusNew,
			gamelift.FleetStatusValidating,
		},
		Target:  []string{gamelift.FleetStatusActive},
		Timeout: 1 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
				FleetIds: aws.StringSlice([]string{d.Id()}),
			})
			if err != nil {
				return 42, "", err
			}

			attributes := out.FleetAttributes
			if len(attributes) < 1 {
				return nil, "", nil
			}
			if len(attributes) != 1 {
				return 42, "", fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
					len(attributes), d.Id())
			}

			fleet := attributes[0]
			return fleet, *fleet.Status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Describing Gamelift Fleet: %s", d.Id())
	out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
		FleetIds: aws.StringSlice([]string{d.Id()}),
	})
	if err != nil {
		return err
	}
	attributes := out.FleetAttributes
	if len(attributes) < 1 {
		log.Printf("[WARN] Gamelift Fleet (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(attributes) != 1 {
		return fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
			len(attributes), d.Id())
	}
	fleet := attributes[0]

	d.Set("build_id", fleet.BuildId)
	d.Set("description", fleet.Description)
	d.Set("fleet_arn", fleet.FleetArn)
	d.Set("log_paths", flattenStringList(fleet.LogPaths))
	d.Set("metric_groups", flattenStringList(fleet.MetricGroups))
	d.Set("name", fleet.Name)
	d.Set("new_game_session_protection_policy", fleet.NewGameSessionProtectionPolicy)
	d.Set("operating_system", fleet.OperatingSystem)
	d.Set("resource_creation_limit_policy", flattenGameliftResourceCreationLimitPolicy(fleet.ResourceCreationLimitPolicy))
	d.Set("server_launch_parameters", fleet.ServerLaunchParameters)
	d.Set("server_launch_path", fleet.ServerLaunchPath)

	return nil
}

func resourceAwsGameliftFleetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Updating Gamelift Fleet: %s", d.Id())
	_, err := conn.UpdateFleetAttributes(&gamelift.UpdateFleetAttributesInput{
		Description:  aws.String(d.Get("description").(string)),
		FleetId:      aws.String(d.Get("fleet_id").(string)),
		MetricGroups: expandStringList(d.Get("metric_groups").([]interface{})),
		Name:         aws.String(d.Get("name").(string)),
		NewGameSessionProtectionPolicy: aws.String(d.Get("new_game_session_protection_policy").(string)),
		ResourceCreationLimitPolicy:    expandGameliftResourceCreationLimitPolicy(d.Get("resource_creation_limit_policy").([]interface{})),
	})
	if err != nil {
		return err
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Deleting Gamelift Fleet: %s", d.Id())
	_, err := conn.DeleteFleet(&gamelift.DeleteFleetInput{
		FleetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	return nil
}

func expandGameliftIpPermissions(cfgs []interface{}) []*gamelift.IpPermission {
	if len(cfgs) < 1 {
		return []*gamelift.IpPermission{}
	}

	perms := make([]*gamelift.IpPermission, len(cfgs), len(cfgs))
	for i, rawCfg := range cfgs {
		cfg := rawCfg.(map[string]interface{})
		perms[i] = &gamelift.IpPermission{
			FromPort: aws.Int64(int64(cfg["from_port"].(int))),
			IpRange:  aws.String(cfg["ip_range"].(string)),
			Protocol: aws.String(cfg["protocol"].(string)),
			ToPort:   aws.Int64(int64(cfg["to_port"].(int))),
		}
	}
	return perms
}

func flattenGameliftIpPermissions(ipps []*gamelift.IpPermission) []interface{} {
	perms := make([]interface{}, len(ipps), len(ipps))

	for i, ipp := range ipps {
		m := make(map[string]interface{}, 0)
		m["from_port"] = *ipp.FromPort
		m["ip_range"] = *ipp.IpRange
		m["protocol"] = *ipp.Protocol
		m["to_port"] = *ipp.ToPort
		perms[i] = m
	}

	return perms
}

func expandGameliftResourceCreationLimitPolicy(cfg []interface{}) *gamelift.ResourceCreationLimitPolicy {
	// TODO
	return nil
}

func flattenGameliftResourceCreationLimitPolicy(policy *gamelift.ResourceCreationLimitPolicy) []interface{} {
	// TODO
	return []interface{}{}
}

func expandGameliftRuntimeConfiguration(cfg []interface{}) *gamelift.RuntimeConfiguration {
	// TODO
	return nil
}
