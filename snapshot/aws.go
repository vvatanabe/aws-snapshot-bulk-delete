package snapshot

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ErrorWithSnapshot struct {
	Error    error
	Snapshot *ec2.Snapshot
}

type awsConfig struct {
	region          string
	profile         string
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	verbose         bool
}

func (c *awsConfig) hasAccessKeys() bool {
	return len(c.accessKeyID) != 0 && len(c.secretAccessKey) != 0
}

func (c *awsConfig) rawAWSConfig() *aws.Config {
	awsCfg := &aws.Config{
		Region: aws.String(c.region),
	}
	if c.verbose {
		awsCfg.CredentialsChainVerboseErrors = aws.Bool(true)
		awsCfg.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	}
	if c.hasAccessKeys() {
		awsCfg.Credentials = credentials.NewStaticCredentials(c.accessKeyID,
			c.secretAccessKey, c.sessionToken)
	}
	return awsCfg
}

func newEC2SnapshotAPI(cfg *awsConfig) (EC2SnapshotAPI, error) {
	sess, err := session.NewSessionWithOptions(newAWSSessionOptions(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to new aws sesion: %w", err)
	}
	return ec2.New(sess), nil
}

func newAWSSessionOptions(cfg *awsConfig) session.Options {
	var opts session.Options
	opts.Config = *cfg.rawAWSConfig()
	opts.Profile = cfg.profile
	opts.SharedConfigState = session.SharedConfigEnable
	return opts
}

type EC2SnapshotAPI interface {
	DescribeSnapshotsPagesWithContext(ctx aws.Context, input *ec2.DescribeSnapshotsInput, fn func(*ec2.DescribeSnapshotsOutput, bool) bool, opts ...request.Option) error
	DeleteSnapshotWithContext(ctx aws.Context, input *ec2.DeleteSnapshotInput, opts ...request.Option) (*ec2.DeleteSnapshotOutput, error)
}
