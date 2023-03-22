package snapshot

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func Test_awsConfig_hasAccessKeys(t *testing.T) {
	type fields struct {
		accessKeyID     string
		secretAccessKey string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "has AccessKeyID and has SecretAccessKey",
			fields: fields{
				accessKeyID:     "foo",
				secretAccessKey: "bar",
			},
			want: true,
		},
		{
			name: "does not have AccessKeyID",
			fields: fields{
				accessKeyID:     "",
				secretAccessKey: "bar",
			},
			want: false,
		},
		{
			name: "does not have SecretAccessKey",
			fields: fields{
				accessKeyID:     "foo",
				secretAccessKey: "",
			},
			want: false,
		},
		{
			name: "does not have both",
			fields: fields{
				accessKeyID:     "",
				secretAccessKey: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &awsConfig{
				accessKeyID:     tt.fields.accessKeyID,
				secretAccessKey: tt.fields.secretAccessKey,
			}
			if got := c.hasAccessKeys(); got != tt.want {
				t.Errorf("hasAccessKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_awsConfig_rawAWSConfig(t *testing.T) {
	type fields struct {
		region          string
		profile         string
		accessKeyID     string
		secretAccessKey string
		sessionToken    string
		verbose         bool
	}
	tests := []struct {
		name   string
		fields fields
		want   *aws.Config
	}{
		{
			name: "profile",
			fields: fields{
				region:  "deadbeef",
				profile: "badfood",
				verbose: false,
			},
			want: &aws.Config{
				Region:                        aws.String("deadbeef"),
				CredentialsChainVerboseErrors: aws.Bool(false),
				LogLevel:                      nil,
				Credentials:                   nil,
			},
		},
		{
			name: "accessKey",
			fields: fields{
				region:          "deadbeef",
				profile:         "",
				accessKeyID:     "badcafe",
				secretAccessKey: "cafebabe",
				sessionToken:    "defecated",
				verbose:         false,
			},
			want: &aws.Config{
				Region:                        aws.String("deadbeef"),
				CredentialsChainVerboseErrors: aws.Bool(false),
				LogLevel:                      nil,
				Credentials:                   credentials.NewStaticCredentials("badcafe", "cafebabe", "defecated"),
			},
		},
		{
			name: "verbose",
			fields: fields{
				region:          "deadbeef",
				profile:         "",
				accessKeyID:     "",
				secretAccessKey: "",
				sessionToken:    "",
				verbose:         true,
			},
			want: &aws.Config{
				Region:                        aws.String("deadbeef"),
				CredentialsChainVerboseErrors: aws.Bool(true),
				LogLevel:                      aws.LogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors),
				Credentials:                   nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &awsConfig{
				region:          tt.fields.region,
				profile:         tt.fields.profile,
				accessKeyID:     tt.fields.accessKeyID,
				secretAccessKey: tt.fields.secretAccessKey,
				sessionToken:    tt.fields.sessionToken,
				verbose:         tt.fields.verbose,
			}
			got := c.rawAWSConfig()
			if !reflect.DeepEqual(*got.Region, *tt.want.Region) {
				t.Errorf("rawAWSConfig() = %v, want %v", *got.Region, *tt.want.Region)
			}
			if got.CredentialsChainVerboseErrors != nil {
				if !reflect.DeepEqual(*got.CredentialsChainVerboseErrors, *tt.want.CredentialsChainVerboseErrors) {
					t.Errorf("rawAWSConfig() = %v, want %v", *got.CredentialsChainVerboseErrors, *tt.want.CredentialsChainVerboseErrors)
				}
			}
			if got.LogLevel != nil {
				if !reflect.DeepEqual(*got.LogLevel, *tt.want.LogLevel) {
					t.Errorf("rawAWSConfig() = %v, want %v", *got.LogLevel, *tt.want.LogLevel)
				}
			}
			if !reflect.DeepEqual(got.Credentials, tt.want.Credentials) {
				t.Errorf("rawAWSConfig() = %v, want %v", got.Credentials, tt.want.Credentials)
			}
		})
	}
}

func Test_tagsMapEC2Filters(t *testing.T) {
	type args struct {
		tags map[string]string
	}
	tests := []struct {
		name string
		args args
		want []*ec2.Filter
	}{
		{
			name: "single value",
			args: args{
				tags: map[string]string{
					"Name": "foo",
				},
			},
			want: []*ec2.Filter{
				{
					Name:   aws.String("tag:Name"),
					Values: []*string{aws.String("foo")},
				},
			},
		},
		{
			name: "multi value",
			args: args{
				tags: map[string]string{
					"Attribute": "bar,baz",
				},
			},
			want: []*ec2.Filter{
				{
					Name:   aws.String("tag:Attribute"),
					Values: []*string{aws.String("bar"), aws.String("baz")},
				},
			},
		},
		{
			name: "trim space",
			args: args{
				tags: map[string]string{
					" Name ":      " foo ",
					" Attribute ": " bar, baz ",
				},
			},
			want: []*ec2.Filter{
				{
					Name:   aws.String("tag:Name"),
					Values: []*string{aws.String("foo")},
				},
				{
					Name:   aws.String("tag:Attribute"),
					Values: []*string{aws.String("bar"), aws.String("baz")},
				},
			},
		},
		{
			name: "empty",
			args: args{
				tags: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagsMapEC2Filters(tt.args.tags)
			if len(got) != len(tt.want) {
				t.Errorf("tagsMapEC2Filters() = %v, want %v", got, tt.want)
			}
			for _, set := range tt.want {
				ok := false
				for _, gotSet := range got {
					if reflect.DeepEqual(gotSet, set) {
						ok = true
						break
					}
				}
				if !ok {
					t.Errorf("tagsMapEC2Filters() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

type ec2SnapshotAPIMock struct {
	DescribeSnapshotsPagesWithContextFunc func(ctx aws.Context, input *ec2.DescribeSnapshotsInput, fn func(*ec2.DescribeSnapshotsOutput, bool) bool, opts ...request.Option) error
	DeleteSnapshotWithContextFunc         func(ctx aws.Context, input *ec2.DeleteSnapshotInput, opts ...request.Option) (*ec2.DeleteSnapshotOutput, error)
}

func (m *ec2SnapshotAPIMock) DescribeSnapshotsPagesWithContext(ctx aws.Context, input *ec2.DescribeSnapshotsInput, fn func(*ec2.DescribeSnapshotsOutput, bool) bool, opts ...request.Option) error {
	return m.DescribeSnapshotsPagesWithContextFunc(ctx, input, fn, opts...)
}

func (m *ec2SnapshotAPIMock) DeleteSnapshotWithContext(ctx aws.Context, input *ec2.DeleteSnapshotInput, opts ...request.Option) (*ec2.DeleteSnapshotOutput, error) {
	return m.DeleteSnapshotWithContextFunc(ctx, input, opts...)
}

func newSnapshot(snapshotId string, startTime time.Time, tagSet []string) *ec2.Snapshot {
	var tags []*ec2.Tag
	for i, v := range tagSet {
		if i%2 == 0 && i+1 < len(tagSet) {
			tags = append(tags, &ec2.Tag{
				Key:   aws.String(v),
				Value: aws.String(tagSet[i+1]),
			})
		}

	}
	return &ec2.Snapshot{
		SnapshotId: aws.String(snapshotId),
		StartTime:  aws.Time(startTime),
		Tags:       tags,
	}
}
