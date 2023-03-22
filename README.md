# aws-snapshot-bulk-delete

Bulk delete AWS EBS Snapshot with tags and expiration date.

## Requires
Go 1.20+

## Installation for library
This package can be installed as library with the go get command:
```
$ go get -u github.com/vvatanabe/aws-snapshot-bulk-delete
```

## Installation for CLI
This package can be installed as CLI with the go install command:
```
$ go install github.com/vvatanabe/aws-snapshot-bulk-delete
```

## Usage for CLI

```
USAGE:
   aws-snapshot-bulk-delete [options]

OPTIONS:
   --region value                                       AWS region [$AWS_REGION]
   --profile value                                      AWS profile [$AWS_PROFILE]
   --access-key-id value                                AWS access key id [$AWS_ACCESS_KEY_ID]
   --secret-access-key value                            AWS secret access key [$AWS_SECRET_ACCESS_KEY]
   --session-token value                                AWS session token [$AWS_SESSION_TOKEN]
   --verbose                                            verbose mode (enable connection debugging) (default: false)
   --plan                                               don't make any changes; instead, try to predict some of the changes that may occur (default: false)
   --age value                                          snapshot retention period (days) (default: 0)
   --tags value [ --tags value ]                        snapshot tags (eg. Name=foo OR Name="foo,bar,baz)"
   --show-properties value [ --show-properties value ]  show properties in stdout (properties: Description, Encrypted, OwnerAlias, OwnerId, Progress, SnapshotId, StartTime, State, StorageTier, VolumeId, VolumeSize, Tags)
   --show-tags value [ --show-tags value ]              show tags in stdout
   --help, -h                                           show help
```

## Usage for Library

### snapshot.BulkDelete#Run

```
package main

import (
	"context"
	"fmt"

	"github.com/vvatanabe/aws-snapshot-bulk-delete/snapshot"
)

func main() {
	bulkDelete, err := snapshot.NewBulkDelete(&snapshot.BulkDeleteConfig{
		Region:          "us-east-1",
		Profile:         "example",
		AccessKeyID:     "foo",
		SecretAccessKey: "bar",
		SessionToken:    "baz",
		Verbose:         false,
		Plan:            false,
		Age:             100,
		Tags:            []string{"Name:foo"},
	})
	if err != nil {
		fmt.Println("failed to init", err)
		return
	}
	err = bulkDelete.Run(context.Background())
	if err != nil {
		fmt.Println("failed to run", err)
		return
	}
}
```

### snapshot.BulkDelete#RunWitOption
```
	err = bulkDelete.RunWithOptions(context.Background(), snapshot.Options{
		BeforeDescribeSnapshotsFunc: func() error {
			return nil
		},
		AfterDescribeSnapshotsFunc: func(snapshots []*ec2.Snapshot) error {
			return nil
		},
		BeforeDeleteSnapshotsFunc: func(snapshots []*ec2.Snapshot) error {
			return nil
		},
		EachDeleteSnapshotsFunc: func(snapshot *ec2.Snapshot) error {
			return nil
		},
		AfterDeleteSnapshotsFunc: func(successful []*ec2.Snapshot, failed []*snapshot.ErrorWithSnapshot) error {
			return nil
		},
	})
	if err != nil {
		fmt.Println("failed to run with option", err)
		return
	}
```


## Bugs and Feedback

For bugs, questions and discussions please use the GitHub Issues.

## License

[MIT License](http://www.opensource.org/licenses/mit-license.php)
