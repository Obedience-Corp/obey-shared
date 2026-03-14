package buildutil

import (
	"slices"
	"testing"
)

func TestBuildTagsArgs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     BuildConfig
		envTags string
		want    []string
	}{
		{
			name: "no tags",
			cfg:  BuildConfig{},
			want: nil,
		},
		{
			name: "config tags only",
			cfg:  BuildConfig{BuildTags: []string{"dev"}},
			want: []string{"-tags", "dev"},
		},
		{
			name: "multiple config tags",
			cfg:  BuildConfig{BuildTags: []string{"dev", "debug"}},
			want: []string{"-tags", "dev,debug"},
		},
		{
			name:    "env tags only",
			cfg:     BuildConfig{},
			envTags: "integration",
			want:    []string{"-tags", "integration"},
		},
		{
			name:    "config and env tags merged",
			cfg:     BuildConfig{BuildTags: []string{"dev"}},
			envTags: "race,smoke",
			want:    []string{"-tags", "dev,race,smoke"},
		},
		{
			name:    "env tags whitespace trimmed",
			cfg:     BuildConfig{},
			envTags: "  ",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envTags != "" {
				t.Setenv("BUILD_TAGS", tt.envTags)
			} else {
				t.Setenv("BUILD_TAGS", "")
			}

			got := buildTagsArgs(tt.cfg)
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildTagsArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegrationTestDir(t *testing.T) {
	tests := []struct {
		name string
		cfg  BuildConfig
		want string
	}{
		{
			name: "default",
			cfg:  BuildConfig{},
			want: "tests/integration",
		},
		{
			name: "custom",
			cfg:  BuildConfig{IntegrationTestDir: "test/e2e"},
			want: "test/e2e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := integrationTestDir(tt.cfg)
			if got != tt.want {
				t.Errorf("integrationTestDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTestOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantPass int
		wantFail int
	}{
		{
			name:     "empty output",
			output:   "",
			wantPass: 0,
			wantFail: 0,
		},
		{
			name: "all passing",
			output: `{"Action":"run","Test":"TestFoo"}
{"Action":"pass","Test":"TestFoo","Elapsed":0.1}
{"Action":"run","Test":"TestBar"}
{"Action":"pass","Test":"TestBar","Elapsed":0.2}
{"Action":"pass","Package":"example.com/pkg"}`,
			wantPass: 2,
			wantFail: 0,
		},
		{
			name: "mixed results",
			output: `{"Action":"run","Test":"TestGood"}
{"Action":"pass","Test":"TestGood","Elapsed":0.1}
{"Action":"run","Test":"TestBad"}
{"Action":"fail","Test":"TestBad","Elapsed":0.3}`,
			wantPass: 1,
			wantFail: 1,
		},
		{
			name: "subtests excluded from count",
			output: `{"Action":"run","Test":"TestParent"}
{"Action":"run","Test":"TestParent/sub1"}
{"Action":"pass","Test":"TestParent/sub1"}
{"Action":"run","Test":"TestParent/sub2"}
{"Action":"fail","Test":"TestParent/sub2"}
{"Action":"fail","Test":"TestParent"}`,
			wantPass: 0,
			wantFail: 1,
		},
		{
			name:     "invalid json ignored",
			output:   "not json\n{\"Action\":\"pass\",\"Test\":\"TestOk\"}\ngarbage",
			wantPass: 1,
			wantFail: 0,
		},
		{
			name: "package-level events ignored",
			output: `{"Action":"pass","Package":"example.com/pkg","Elapsed":1.5}
{"Action":"fail","Package":"example.com/other","Elapsed":2.0}`,
			wantPass: 0,
			wantFail: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, fail := parseTestOutput([]byte(tt.output), false)
			if pass != tt.wantPass {
				t.Errorf("passed = %d, want %d", pass, tt.wantPass)
			}
			if fail != tt.wantFail {
				t.Errorf("failed = %d, want %d", fail, tt.wantFail)
			}
		})
	}
}

