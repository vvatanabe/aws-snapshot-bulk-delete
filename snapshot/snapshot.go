package snapshot

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type BulkDeleteConfig struct {
	Region          string
	Profile         string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Verbose         bool

	Plan bool

	Age  uint
	Tags []string
}

func (cfg *BulkDeleteConfig) hasAgeOrTags() bool {
	return cfg.Age > 0 || len(cfg.Tags) > 0
}

func (cfg *BulkDeleteConfig) awsConfig() *awsConfig {
	return &awsConfig{
		region:          cfg.Region,
		profile:         cfg.Profile,
		accessKeyID:     cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
		sessionToken:    cfg.SessionToken,
		verbose:         cfg.Verbose,
	}
}

func NewBulkDelete(cfg *BulkDeleteConfig) (*BullDelete, error) {
	if !cfg.hasAgeOrTags() {
		return nil, fmt.Errorf("both age and tags not specified")
	}
	tags, err := tagsMap(cfg.Tags)
	if err != nil {
		return nil, err
	}
	client, err := newEC2SnapshotAPI(cfg.awsConfig())
	if err != nil {
		return nil, err
	}
	return &BullDelete{
		age:  cfg.Age,
		tags: tags,
		plan: cfg.Plan,
		svc:  client,
	}, nil
}

func tagsMap(tags []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, kv := range tags {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" || v == "" {
			return m, fmt.Errorf("invalid tag: %s", kv)
		}
		m[k] = v
	}
	return m, nil
}

type BullDelete struct {
	age  uint
	tags map[string]string
	plan bool
	svc  EC2SnapshotAPI
}

type Options struct {
	BeforeDescribeSnapshotsFunc func() error
	AfterDescribeSnapshotsFunc  func(snapshots []*ec2.Snapshot) error
	BeforeDeleteSnapshotsFunc   func(snapshots []*ec2.Snapshot) error
	EachDeleteSnapshotsFunc     func(snapshot *ec2.Snapshot) error
	AfterDeleteSnapshotsFunc    func(successful []*ec2.Snapshot, failed []*ErrorWithSnapshot) error
}

func (c *BullDelete) Run(ctx context.Context) error {
	return c.RunWithOptions(ctx, Options{})
}

func (c *BullDelete) RunWithOptions(ctx context.Context, opts Options) error {
	ctx = setNow(ctx)
	if opts.BeforeDescribeSnapshotsFunc != nil {
		err := opts.BeforeDescribeSnapshotsFunc()
		if err != nil {
			return err
		}
	}

	snapshots, err := c.describeSnapshots(ctx, c.tags, c.age)
	if err != nil {
		return err
	}

	if opts.AfterDescribeSnapshotsFunc != nil {
		err := opts.AfterDescribeSnapshotsFunc(snapshots)
		if err != nil {
			return err
		}
	}

	if c.plan {
		return nil
	}

	if opts.BeforeDeleteSnapshotsFunc != nil {
		err := opts.BeforeDeleteSnapshotsFunc(snapshots)
		if err != nil {
			return err
		}
	}

	successful, failed, err := c.deleteSnapshots(ctx, snapshots, opts.EachDeleteSnapshotsFunc)
	if err != nil {
		return err
	}

	if opts.AfterDeleteSnapshotsFunc != nil {
		err := opts.AfterDeleteSnapshotsFunc(successful, failed)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *BullDelete) describeSnapshots(ctx context.Context, tags map[string]string, age uint) ([]*ec2.Snapshot, error) {
	var snapshots []*ec2.Snapshot
	err := c.svc.DescribeSnapshotsPagesWithContext(ctx, &ec2.DescribeSnapshotsInput{
		Filters: tagsMapEC2Filters(tags),
	}, func(out *ec2.DescribeSnapshotsOutput, lastPage bool) bool {
		snapshots = append(snapshots, out.Snapshots...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	if age > 0 {
		expireDate := now(ctx).Add(-time.Duration(age) * 24 * time.Hour)
		snapshots = filterSnapshots(snapshots, expiredFilterFunc(expireDate))
	}
	if len(tags) > 0 {
		snapshots = filterSnapshots(snapshots, tagsFilterFunc(tags))
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].StartTime.Before(*snapshots[j].StartTime)
	})
	return snapshots, nil
}

func (c *BullDelete) deleteSnapshots(ctx context.Context,
	snapshots []*ec2.Snapshot, eachFunc func(snapshot *ec2.Snapshot) error) ([]*ec2.Snapshot, []*ErrorWithSnapshot, error) {
	var (
		successful []*ec2.Snapshot
		failed     []*ErrorWithSnapshot
	)
	for _, snapshot := range snapshots {
		_, err := c.svc.DeleteSnapshotWithContext(ctx, &ec2.DeleteSnapshotInput{
			SnapshotId: snapshot.SnapshotId,
		})
		if err != nil {
			failed = append(failed, &ErrorWithSnapshot{Snapshot: snapshot, Error: err})
			continue
		}
		successful = append(successful, snapshot)
		if eachFunc != nil {
			err := eachFunc(snapshot)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return successful, failed, nil
}

func tagsMapEC2Filters(tags map[string]string) []*ec2.Filter {
	var filters []*ec2.Filter
	if tags == nil {
		return filters
	}
	for k, v := range tags {
		k = strings.TrimSpace(k)
		filter := &ec2.Filter{
			Name: aws.String(fmt.Sprintf("tag:%s", k)),
		}
		for _, v := range strings.Split(v, ",") {
			v = strings.TrimSpace(v)
			filter.Values = append(filter.Values, aws.String(v))
		}
		filters = append(filters, filter)
	}
	return filters
}

func filterSnapshots(snapshots []*ec2.Snapshot, fn filterFunc) []*ec2.Snapshot {
	var matches []*ec2.Snapshot
	for _, snapshot := range snapshots {
		if fn(snapshot) {
			matches = append(matches, snapshot)
		}
	}
	return matches
}

type filterFunc = func(*ec2.Snapshot) bool

func expiredFilterFunc(expireDate time.Time) filterFunc {
	return func(snapshot *ec2.Snapshot) bool {
		return snapshot.StartTime.Before(expireDate)
	}
}

func tagsFilterFunc(tags map[string]string) filterFunc {
	return func(snapshot *ec2.Snapshot) bool {
		for _, tag := range snapshot.Tags {
			if v, ok := tags[aws.StringValue(tag.Key)]; ok && v != "" {
				tagValues := strings.Split(aws.StringValue(tag.Value), ",")
				for _, tagValue := range tagValues {
					if v == tagValue {
						return true
					}
				}
			}
		}
		return false
	}
}
