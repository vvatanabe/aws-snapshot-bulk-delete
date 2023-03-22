package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cheggaaa/pb/v3"

	"github.com/manifoldco/promptui"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/urfave/cli/v2"
	"github.com/vvatanabe/aws-snapshot-bulk-delete/snapshot"
)

const (
	appName     = "aws-snapshot-bulk-delete"
	usage       = "Bulk delete AWS EBS snapshot"
	description = "Bulk delete AWS EBS snapshot with tags and expiration date."

	flagNameRegion          = "region"
	flagNameProfile         = "profile"
	flagNameAccessKeyID     = "access-key-id"
	flagNameSecretAccessKey = "secret-access-key"
	flagNameSessionToken    = "session-token"
	flagNameVerbose         = "verbose"
	flagNamePlan            = "plan"
	flagNameAge             = "age"
	flagNameTags            = "tags"
	flagShowProperties      = "show-properties"
	flagShowTags            = "show-tags"
)

var defaultProperties = []string{
	"Description",
	"Encrypted",
	"OwnerAlias",
	"OwnerId",
	"Progress",
	"SnapshotId",
	"StartTime",
	"State",
	"StorageTier",
	"VolumeId",
	"VolumeSize",
	"Tags",
}

var defaultPropertiesSet = make(map[string]struct{})

func init() {
	for _, v := range defaultProperties {
		defaultPropertiesSet[v] = struct{}{}
	}
}

func toEnvVarCase(prefix, name string) string {
	// eg. foo-bar-baz to PREFIX_FOO_BAR_BAZ
	if prefix != "" {
		name = prefix + "_" + name
	}
	name = strings.ToUpper(name)
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

func app() *cli.App {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = usage
	app.Description = description
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     flagNameRegion,
			EnvVars:  []string{toEnvVarCase("AWS", flagNameRegion)},
			Usage:    "AWS region",
			Required: true,
		},
		&cli.StringFlag{
			Name:    flagNameProfile,
			EnvVars: []string{toEnvVarCase("AWS", flagNameProfile)},
			Usage:   "AWS profile",
		},
		&cli.StringFlag{
			Name:    flagNameAccessKeyID,
			EnvVars: []string{toEnvVarCase("AWS", flagNameAccessKeyID)},
			Usage:   "AWS access key id",
		},
		&cli.StringFlag{
			Name:    flagNameSecretAccessKey,
			EnvVars: []string{toEnvVarCase("AWS", flagNameSecretAccessKey)},
			Usage:   "AWS secret access key",
		},
		&cli.StringFlag{
			Name:    flagNameSessionToken,
			EnvVars: []string{toEnvVarCase("AWS", flagNameSessionToken)},
			Usage:   "AWS session token",
		},
		&cli.BoolFlag{
			Name:  flagNameVerbose,
			Usage: "verbose mode (enable connection debugging)",
		},
		&cli.BoolFlag{
			Name:  flagNamePlan,
			Usage: "don't make any changes; instead, try to predict some of the changes that may occur",
		},
		&cli.UintFlag{
			Name:  flagNameAge,
			Usage: "snapshot retention period (days)",
		},
		&cli.StringSliceFlag{
			Name:  flagNameTags,
			Usage: "snapshot tags (eg. Name=foo OR Name=\"foo,bar,baz)\"",
		},
		&cli.StringSliceFlag{
			Name:  flagShowProperties,
			Usage: "show properties in stdout (properties: Description, Encrypted, OwnerAlias, OwnerId, Progress, SnapshotId, StartTime, State, StorageTier, VolumeId, VolumeSize, Tags)",
		},
		&cli.StringSliceFlag{
			Name:  flagShowTags,
			Usage: "show tags in stdout",
		},
	}
	app.Action = action
	return app
}

func action(c *cli.Context) error {
	cfg := parseConfig(c)
	bulkDelete, err := snapshot.NewBulkDelete(cfg)
	if err != nil {
		return err
	}
	showPropertiesSet := initShowPropertiesSet(c.StringSlice(flagShowProperties))
	showTagsSet := initShowTagsSet(c.StringSlice(flagShowTags))
	var bar *pb.ProgressBar
	return bulkDelete.RunWithOptions(context.Background(), snapshot.Options{
		AfterDescribeSnapshotsFunc: func(snapshots []*ec2.Snapshot) error {
			writeSnapshotDeletionPlan(os.Stdout, snapshots, showPropertiesSet, showTagsSet)
			if cfg.Plan {
				return nil
			}
			writeConfirmMessage(os.Stdout)
			return runConfirmPrompt()
		},
		BeforeDeleteSnapshotsFunc: func(snapshots []*ec2.Snapshot) error {
			bar = pb.StartNew(len(snapshots))
			return nil
		},
		EachDeleteSnapshotsFunc: func(snapshot *ec2.Snapshot) error {
			bar.Increment()
			return nil
		},
		AfterDeleteSnapshotsFunc: func(successful []*ec2.Snapshot, failed []*snapshot.ErrorWithSnapshot) error {
			bar.Finish()
			writeSnapshotDeletionResult(os.Stdout, successful, failed, showPropertiesSet, showTagsSet)
			return nil
		},
	})
}

func parseConfig(c *cli.Context) *snapshot.BulkDeleteConfig {
	return &snapshot.BulkDeleteConfig{
		Region:          c.String(flagNameRegion),
		Profile:         c.String(flagNameProfile),
		AccessKeyID:     c.String(flagNameAccessKeyID),
		SecretAccessKey: c.String(flagNameSecretAccessKey),
		SessionToken:    c.String(flagNameSessionToken),
		Verbose:         c.Bool(flagNameVerbose),
		Plan:            c.Bool(flagNamePlan),
		Age:             c.Uint(flagNameAge),
		Tags:            c.StringSlice(flagNameTags),
	}
}

func initShowPropertiesSet(showProperties []string) map[string]struct{} {
	if len(showProperties) == 0 {
		return defaultPropertiesSet
	}
	set := make(map[string]struct{})
	for _, v := range showProperties {
		set[strings.TrimSpace(v)] = struct{}{}
	}
	return set
}

func initShowTagsSet(showTags []string) map[string]struct{} {
	tags := make(map[string]struct{})
	for _, v := range showTags {
		tags[strings.TrimSpace(v)] = struct{}{}
	}
	return tags
}

func runConfirmPrompt() error {
	prompt := promptui.Prompt{
		Label:     "Enter a value",
		IsConfirm: true,
	}
	_, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("\nApply cancelled.")
	}
	return nil
}

func writeConfirmMessage(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Do you want to perform this actions?
%s will perform the actions described above.
Only 'y' will be accepted to approve.

`, appName)
}

func writeSnapshotDeletionPlan(w io.Writer, snapshots []*ec2.Snapshot, showPropertiesSet map[string]struct{}, showTagsSet map[string]struct{}) {
	tw := tabwriter.NewWriter(w, 0, 1, 4, ' ', tabwriter.TabIndent)
	headerLine := buildHeaderLine(showPropertiesSet)
	_, _ = tw.Write([]byte(headerLine + "\t\n"))
	for _, v := range snapshots {
		line := buildPropertiesLine(v, showPropertiesSet, showTagsSet)
		_, _ = tw.Write([]byte(line + "\t\n"))
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintf(w, "\n")
	_, _ = fmt.Fprintf(w, "Plan: %d to delete.\n\n", len(snapshots))
}

func writeSnapshotDeletionResult(w io.Writer, successful []*ec2.Snapshot, failed []*snapshot.ErrorWithSnapshot, showPropertiesSet map[string]struct{}, showTagsSet map[string]struct{}) {
	tw := tabwriter.NewWriter(w, 0, 1, 4, ' ', tabwriter.TabIndent)
	headerLine := buildHeaderLine(showPropertiesSet)
	_, _ = tw.Write([]byte("Result\t" + headerLine + "error\t\n"))
	for _, v := range successful {
		line := buildPropertiesLine(v, showPropertiesSet, showTagsSet)
		_, _ = tw.Write([]byte("successful\t" + line + "-\t\n"))
	}
	for _, v := range failed {
		line := buildPropertiesLine(v.Snapshot, showPropertiesSet, showTagsSet)
		_, _ = tw.Write([]byte("failed\t" + line + v.Error.Error() + "\t\n"))
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintf(w, "\n")
	_, _ = fmt.Fprintf(w, "Result: %d to successful, %d to failed.\n\n", len(successful), len(failed))
}

func buildHeaderLine(showPropertiesSet map[string]struct{}) string {
	var b strings.Builder
	for _, dp := range defaultProperties {
		if _, ok := showPropertiesSet[dp]; ok {
			b.WriteString(dp + "\t")
		}
	}
	return b.String()
}

func buildTagsLine(tags []*ec2.Tag, showTagsSet map[string]struct{}) string {
	tagsMap := make(map[string]string)
	for _, tag := range tags {
		key := aws.StringValue(tag.Key)
		value := aws.StringValue(tag.Value)
		if _, ok := showTagsSet[key]; ok || len(showTagsSet) == 0 {
			tagsMap["\""+key+"\""] = "\"" + value + "\""
		}
	}
	tagsLine := fmt.Sprintf("%v", tagsMap)
	tagsLine = strings.TrimPrefix(tagsLine, "map[")
	tagsLine = strings.TrimSuffix(tagsLine, "]")
	return tagsLine
}

func buildPropertiesLine(snapshot *ec2.Snapshot, showPropertiesSet map[string]struct{}, showTagsSet map[string]struct{}) string {
	var b strings.Builder
	for _, dp := range defaultProperties {
		if _, ok := showPropertiesSet[dp]; !ok {
			continue
		}
		switch dp {
		case "Description":
			b.WriteString(aws.StringValue(snapshot.Description))
		case "Encrypted":
			b.WriteString(strconv.FormatBool(aws.BoolValue(snapshot.Encrypted)))
		case "OwnerAlias":
			b.WriteString(aws.StringValue(snapshot.OwnerAlias))
		case "OwnerId":
			b.WriteString(aws.StringValue(snapshot.OwnerId))
		case "Progress":
			b.WriteString(aws.StringValue(snapshot.Progress))
		case "SnapshotId":
			b.WriteString(aws.StringValue(snapshot.SnapshotId))
		case "StartTime":
			b.WriteString(aws.TimeValue(snapshot.StartTime).Format(time.RFC3339))
		case "State":
			b.WriteString(aws.StringValue(snapshot.State))
		case "StorageTier":
			b.WriteString(aws.StringValue(snapshot.StorageTier))
		case "VolumeId":
			b.WriteString(aws.StringValue(snapshot.VolumeId))
		case "VolumeSize":
			b.WriteString(strconv.FormatInt(aws.Int64Value(snapshot.VolumeSize), 10))
		case "Tags":
			b.WriteString(buildTagsLine(snapshot.Tags, showTagsSet))
		}
		b.WriteString("\t")
	}
	return b.String()
}
