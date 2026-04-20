package buildutil

import (
	"bytes"
	"io"
	"os"
	"reflect"
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

func TestBuildConfigIsLibrary(t *testing.T) {
	tests := []struct {
		name string
		cfg  BuildConfig
		want bool
	}{
		{
			name: "empty config is library",
			cfg:  BuildConfig{},
			want: true,
		},
		{
			name: "only section name is library",
			cfg:  BuildConfig{SectionName: "obey-shared"},
			want: true,
		},
		{
			name: "binary + main is application",
			cfg:  BuildConfig{BinaryName: "obey", MainPath: "./cmd/obey"},
			want: false,
		},
		{
			name: "binary only is application",
			cfg:  BuildConfig{BinaryName: "obey"},
			want: false,
		},
		{
			name: "main only is application",
			cfg:  BuildConfig{MainPath: "./cmd/obey"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsLibrary(); got != tt.want {
				t.Errorf("IsLibrary() = %v, want %v", got, tt.want)
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

func TestCommandSurfacePath(t *testing.T) {
	tests := []struct {
		name string
		cfg  BuildConfig
		want string
	}{
		{
			name: "defaults to main path",
			cfg:  BuildConfig{MainPath: "./cmd/example"},
			want: "./cmd/example",
		},
		{
			name: "explicit command surface path wins",
			cfg: BuildConfig{
				MainPath:           "./cmd/example",
				CommandSurfacePath: "./cmd/example-profile",
			},
			want: "./cmd/example-profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandSurfacePath(tt.cfg); got != tt.want {
				t.Errorf("commandSurfacePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommandSurfaceArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  BuildConfig
		want []string
	}{
		{
			name: "default helper args",
			cfg:  BuildConfig{},
			want: []string{"__commands"},
		},
		{
			name: "explicit helper args",
			cfg:  BuildConfig{CommandSurfaceArgs: []string{"debug", "commands"}},
			want: []string{"debug", "commands"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandSurfaceArgs(tt.cfg); !slices.Equal(got, tt.want) {
				t.Errorf("commandSurfaceArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandSurfaceProfiles(t *testing.T) {
	tests := []struct {
		name    string
		cfg     BuildConfig
		want    []CommandSurfaceProfile
		wantErr string
	}{
		{
			name: "defaults to stable and dev",
			cfg:  BuildConfig{},
			want: []CommandSurfaceProfile{
				{Name: "stable"},
				{Name: "dev", Tags: []string{"dev"}},
			},
		},
		{
			name: "custom profiles preserved",
			cfg: BuildConfig{
				CommandSurfaceProfiles: []CommandSurfaceProfile{
					{Name: "stable"},
					{Name: "enterprise", Tags: []string{"enterprise"}},
					{Name: "experimental", Tags: []string{"exp", "debug"}},
				},
			},
			want: []CommandSurfaceProfile{
				{Name: "stable"},
				{Name: "enterprise", Tags: []string{"enterprise"}},
				{Name: "experimental", Tags: []string{"exp", "debug"}},
			},
		},
		{
			name: "blank profile names rejected",
			cfg: BuildConfig{
				CommandSurfaceProfiles: []CommandSurfaceProfile{
					{Name: "stable"},
					{Name: "   "},
				},
			},
			wantErr: `command surface profiles require a non-empty name`,
		},
		{
			name: "duplicate names rejected",
			cfg: BuildConfig{
				CommandSurfaceProfiles: []CommandSurfaceProfile{
					{Name: "stable"},
					{Name: "stable", Tags: []string{"dev"}},
				},
			},
			wantErr: `duplicate command surface profile "stable"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := commandSurfaceProfiles(tt.cfg)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("commandSurfaceProfiles() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("commandSurfaceProfiles() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("commandSurfaceProfiles() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestProfileCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     BuildConfig
		profile CommandSurfaceProfile
		envTags string
		want    []string
	}{
		{
			name: "stable uses configured tags and defaults",
			cfg: BuildConfig{
				MainPath:  "./cmd/example",
				BuildTags: []string{"release"},
			},
			profile: CommandSurfaceProfile{Name: "stable"},
			want:    []string{"run", "-tags", "release", "./cmd/example", "__commands"},
		},
		{
			name: "profile appends extra tags",
			cfg: BuildConfig{
				MainPath:  "./cmd/example",
				BuildTags: []string{"release"},
			},
			profile: CommandSurfaceProfile{Name: "dev", Tags: []string{"dev"}},
			want:    []string{"run", "-tags", "release,dev", "./cmd/example", "__commands"},
		},
		{
			name: "custom surface path and args",
			cfg: BuildConfig{
				MainPath:           "./cmd/example",
				CommandSurfacePath: "./cmd/surface",
				CommandSurfaceArgs: []string{"debug", "commands"},
			},
			profile: CommandSurfaceProfile{Name: "dev", Tags: []string{"dev"}},
			want:    []string{"run", "-tags", "dev", "./cmd/surface", "debug", "commands"},
		},
		{
			name: "env tags are included",
			cfg: BuildConfig{
				MainPath: "./cmd/example",
			},
			profile: CommandSurfaceProfile{Name: "enterprise", Tags: []string{"enterprise"}},
			envTags: "trace",
			want:    []string{"run", "-tags", "enterprise,trace", "./cmd/example", "__commands"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BUILD_TAGS", tt.envTags)

			if got := profileCommandArgs(tt.cfg, tt.profile); !slices.Equal(got, tt.want) {
				t.Errorf("profileCommandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCommandSurfaceOutput(t *testing.T) {
	output := []byte("\ncmd alpha\n\n  cmd beta  \ncmd gamma\n")

	got := parseCommandSurfaceOutput(output)
	want := []string{"cmd alpha", "cmd beta", "cmd gamma"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCommandSurfaceOutput() = %#v, want %#v", got, want)
	}
}

func TestDiffCommandSurfaces(t *testing.T) {
	base := []string{"cmd alpha", "cmd beta"}
	candidate := []string{"cmd alpha", "cmd beta", "cmd explore", "cmd explore active"}

	got := diffCommandSurfaces(base, candidate)
	want := []string{"cmd explore", "cmd explore active"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diffCommandSurfaces() = %#v, want %#v", got, want)
	}
}

func TestFindCommandSurfaceProfile(t *testing.T) {
	profiles := []CommandSurfaceProfile{
		{Name: "stable"},
		{Name: "enterprise", Tags: []string{"enterprise"}},
	}

	got, ok := findCommandSurfaceProfile(profiles, "enterprise")
	if !ok {
		t.Fatal("expected profile lookup to succeed")
	}
	want := CommandSurfaceProfile{Name: "enterprise", Tags: []string{"enterprise"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findCommandSurfaceProfile() = %#v, want %#v", got, want)
	}

	if _, ok := findCommandSurfaceProfile(profiles, "missing"); ok {
		t.Fatal("expected missing profile lookup to fail")
	}
}

func TestCommandSurfaceProfileNames(t *testing.T) {
	profiles := []CommandSurfaceProfile{
		{Name: "stable"},
		{Name: "dev"},
		{Name: "enterprise"},
	}

	got := commandSurfaceProfileNames(profiles)
	want := []string{"stable", "dev", "enterprise"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commandSurfaceProfileNames() = %#v, want %#v", got, want)
	}
}

func TestPrintCommandProfile(t *testing.T) {
	output := captureStdout(t, func() {
		printCommandProfile("stable", []string{"cmd alpha", "cmd beta"})
	})

	want := "== stable profile (2 commands) ==\ncmd alpha\ncmd beta\n"
	if output != want {
		t.Fatalf("printCommandProfile() = %q, want %q", output, want)
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w

	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy(): %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}

	return buf.String()
}
