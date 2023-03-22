package main

import (
	"flag"
	"reflect"
	"testing"

	"github.com/vvatanabe/aws-snapshot-bulk-delete/snapshot"

	"github.com/urfave/cli/v2"
)

func Test_app(t *testing.T) {
	got := app()
	{
		want := "aws-snapshot-bulk-delete"
		if got.Name != want {
			t.Errorf("app() = %v, want %v", got, want)
		}
	}
	{
		want := []cli.Flag{
			&cli.StringFlag{
				Name:     "region",
				EnvVars:  []string{"AWS_REGION"},
				Usage:    "AWS region",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "profile",
				EnvVars: []string{"AWS_PROFILE"},
				Usage:   "AWS profile",
			},
			&cli.StringFlag{
				Name:    "access-key-id",
				EnvVars: []string{"AWS_ACCESS_KEY_ID"},
				Usage:   "AWS access key id",
			},
			&cli.StringFlag{
				Name:    "secret-access-key",
				EnvVars: []string{"AWS_SECRET_ACCESS_KEY"},
				Usage:   "AWS secret access key",
			},
			&cli.StringFlag{
				Name:    "session-token",
				EnvVars: []string{"AWS_SESSION_TOKEN"},
				Usage:   "AWS session token",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose mode (enable connection debugging)",
			},
			&cli.BoolFlag{
				Name:  "plan",
				Usage: "don't make any changes; instead, try to predict some of the changes that may occur",
			},
			&cli.UintFlag{
				Name:  "age",
				Usage: "snapshot retention period (days)",
			},
			&cli.StringSliceFlag{
				Name:  "tags",
				Usage: "snapshot tags (eg. Name=foo OR Name=\"foo,bar,baz)\"",
			},
			&cli.StringSliceFlag{
				Name:  "show-properties",
				Usage: "show properties in stdout (properties: Description, Encrypted, OwnerAlias, OwnerId, Progress, SnapshotId, StartTime, State, StorageTier, VolumeId, VolumeSize, Tags)",
			},
			&cli.StringSliceFlag{
				Name:  "show-tags",
				Usage: "show tags in stdout",
			},
		}
		if len(got.Flags) != len(want) {
			t.Errorf("got %d, want %d", len(got.Flags), len(want))
		}
		for i, v := range want {
			if !reflect.DeepEqual(got.Flags[i], v) {
				t.Errorf("got = %v, want %v", got.Flags[i], v)
			}
		}
	}

}

func Test_toEnvVarCase(t *testing.T) {
	type args struct {
		prefix string
		name   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "none prefix",
			args: args{
				prefix: "",
				name:   "foo-bar-baz",
			},
			want: "FOO_BAR_BAZ",
		},
		{
			name: "with prefix",
			args: args{
				prefix: "head",
				name:   "foo-bar-baz",
			},
			want: "HEAD_FOO_BAR_BAZ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toEnvVarCase(tt.args.prefix, tt.args.name); got != tt.want {
				t.Errorf("toEnvVarCase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseConfig(t *testing.T) {
	type args struct {
		c *cli.Context
	}
	tests := []struct {
		name string
		args args
		want *snapshot.BulkDeleteConfig
	}{
		{
			name: "walkthrough",
			args: args{
				c: cli.NewContext(app(), &flag.FlagSet{}, nil),
			},
			want: &snapshot.BulkDeleteConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseConfig(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initShowPropertiesSet(t *testing.T) {
	type args struct {
		showProperties []string
	}
	tests := []struct {
		name string
		args args
		want map[string]struct{}
	}{
		{
			name: "default",
			args: args{
				showProperties: []string{},
			},
			want: defaultPropertiesSet,
		},
		{
			name: "some tags",
			args: args{
				showProperties: []string{
					"Description ",
					" SnapshotId",
					" StartTime ",
					"  Tags  ",
				},
			},
			want: map[string]struct{}{
				"Description": {},
				"SnapshotId":  {},
				"StartTime":   {},
				"Tags":        {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initShowPropertiesSet(tt.args.showProperties)
			if len(got) != len(tt.want) {
				t.Errorf("initShowPropertiesSet() = %v, want %v", got, tt.want)
			}
		})
	}
}
