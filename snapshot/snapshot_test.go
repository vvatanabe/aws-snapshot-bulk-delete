package snapshot

import (
	"reflect"
	"testing"
)

func TestConfig_hasAgeOrTags(t *testing.T) {
	type fields struct {
		Age  uint
		Tags []string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "has age",
			fields: fields{
				Age:  10,
				Tags: nil,
			},
			want: true,
		},
		{
			name: "has tags",
			fields: fields{
				Age:  0,
				Tags: []string{"Name=foo"},
			},
			want: true,
		},
		{
			name: "has both",
			fields: fields{
				Age:  10,
				Tags: []string{"Name=foo"},
			},
			want: true,
		},
		{
			name: "doesn't have both",
			fields: fields{
				Age:  0,
				Tags: []string{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BulkDeleteConfig{
				Age:  tt.fields.Age,
				Tags: tt.fields.Tags,
			}
			if got := cfg.hasAgeOrTags(); got != tt.want {
				t.Errorf("hasAgeOrTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_awsConfig(t *testing.T) {
	cfg := &BulkDeleteConfig{
		Region:          "deadbeef",
		Profile:         "badfood",
		AccessKeyID:     "badcafe",
		SecretAccessKey: "cafebabe",
		SessionToken:    "defecated",
		Verbose:         true,
	}
	want := &awsConfig{
		region:          cfg.Region,
		profile:         cfg.Profile,
		accessKeyID:     cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
		sessionToken:    cfg.SessionToken,
		verbose:         cfg.Verbose,
	}
	if got := cfg.awsConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("awsConfig() = %v, want %v", got, want)
	}
}

func Test_tagsMap(t *testing.T) {
	type args struct {
		tags []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "single value",
			args: args{
				tags: []string{"Name=foo"},
			},
			want: map[string]string{
				"Name": "foo",
			},
			wantErr: false,
		},
		{
			name: "multi value",
			args: args{
				tags: []string{"Name=foo", "Attribute=bar"},
			},
			want: map[string]string{
				"Name":      "foo",
				"Attribute": "bar",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			args: args{
				tags: []string{"=foo"},
			},
			want:    map[string]string{},
			wantErr: true,
		},
		{
			name: "empty value",
			args: args{
				tags: []string{"Name="},
			},
			want:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tagsMap(tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("tagsMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tagsMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBulkDelete(t *testing.T) {
	type args struct {
		cfg *BulkDeleteConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *BullDelete
		wantErr bool
	}{
		{
			name: "has age",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  10,
					Tags: []string{},
					Plan: false,
				},
			},
			want: &BullDelete{
				age:  10,
				tags: map[string]string{},
				plan: false,
			},
			wantErr: false,
		},
		{
			name: "has tags",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  0,
					Tags: []string{"Name=foo"},
					Plan: false,
				},
			},
			want: &BullDelete{
				age:  0,
				tags: map[string]string{"Name": "foo"},
				plan: false,
			},
			wantErr: false,
		},
		{
			name: "has age and has tags",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  10,
					Tags: []string{"Name=foo"},
					Plan: false,
				},
			},
			want: &BullDelete{
				age:  10,
				tags: map[string]string{"Name": "foo"},
				plan: false,
			},
			wantErr: false,
		},
		{
			name: "doesn't have age and doesn't have tags",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  0,
					Tags: []string{},
					Plan: false,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid tags",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  0,
					Tags: []string{"Name="},
					Plan: false,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "plan is true",
			args: args{
				cfg: &BulkDeleteConfig{
					Age:  10,
					Tags: []string{"Name=foo"},
					Plan: true,
				},
			},
			want: &BullDelete{
				age:  10,
				tags: map[string]string{"Name": "foo"},
				plan: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBulkDelete(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBulkDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if got != nil {
					t.Errorf("NewBulkDelete() got = %v, want %v", got, nil)
					return
				}
				return
			}
			if !reflect.DeepEqual(got.age, tt.want.age) {
				t.Errorf("NewBulkDelete() got = %v, want %v", got.age, tt.want.age)
			}
			if !reflect.DeepEqual(got.tags, tt.want.tags) {
				t.Errorf("NewBulkDelete() got = %v, want %v", got.tags, tt.want.tags)
			}
			if !reflect.DeepEqual(got.plan, tt.want.plan) {
				t.Errorf("NewBulkDelete() got = %v, want %v", got.plan, tt.want.plan)
			}
		})
	}
}
