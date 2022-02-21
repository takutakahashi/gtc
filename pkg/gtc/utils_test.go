package gtc

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestClient_CommitFiles(t *testing.T) {
	type args struct {
		files   map[string][]byte
		message string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:   "ok_single_file",
			client: mockInit(),
			args: args{
				files: map[string][]byte{
					"test.txt": []byte("123"),
				},
			},
			asserts: map[string][]string{
				"diff": {"test.txt", ""},
			},
			wantErr: false,
		},
		{
			name:   "ok_multiple_files",
			client: mockInit(),
			args: args{
				files: map[string][]byte{
					"test.txt":  []byte("123"),
					"test2.txt": []byte("456"),
					"test3.txt": []byte("789"),
				},
			},
			asserts: map[string][]string{
				"diff": {"test.txt", "test2.txt", "test3.txt", ""},
			},
			wantErr: false,
		},
		{
			name:   "ok_with_dir",
			client: mockInit(),
			args: args{
				files: map[string][]byte{
					"dir/test.txt": []byte("123"),
				},
			},
			asserts: map[string][]string{
				"diff": {"dir/test.txt", ""},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.CommitFiles(tt.args.files, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Client.CommitFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_GetHash(t *testing.T) {
	type args struct {
		base string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		wantErr bool
	}{
		{
			name:   "ok",
			client: mockInit(),
			args: args{
				base: "master",
			},
			wantErr: false,
		},
		{
			name:   "ok_tag",
			client: mockGtc(),
			args: args{
				base: "v0.1.0",
			},
			wantErr: false,
		},
		{
			name:   "ok_revision",
			client: mockGtc(),
			args: args{
				base: "6cac01a031dd3e38ed7fcb12bf6e4e4c08c0b3d7",
			},
			wantErr: false,
		},
		{
			name:   "ng_revision",
			client: mockInit(),
			args: args{
				base: "6cac01a031dd3e38ed7fcb12bf6e4e4c08c0b3d7",
			},
			wantErr: true,
		},
		{
			name:   "ng_norev",
			client: mockInit(),
			args: args{
				base: "no-rev",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.GetHash(tt.args.base)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				return
			}
			if out, err := c.gitExec([]string{"rev-parse", tt.args.base}); err != nil || strings.Compare(out[0], got) != 0 {
				t.Errorf("wrong revision. expected: %s, actual: %s", got, out[0])
				return
			}
		})
	}
}

func TestClient_GetLatestTagReference(t *testing.T) {
	tests := []struct {
		name        string
		client      Client
		wantTagName string
		want        *plumbing.Reference
		wantErr     bool
	}{
		{
			name:        "ok_multiple_tag",
			client:      mockWithTags([]string{"test1", "test3", "test2"}),
			wantTagName: "test2",
			wantErr:     false,
		},
		{
			name:        "ok_single_tag",
			client:      mockWithTags([]string{"test1"}),
			wantTagName: "test1",
			wantErr:     false,
		},
		{
			name:    "ng_no_tag",
			client:  mockInit(),
			wantErr: true,
		},
		{
			name:    "ng_only_remote_tag",
			client:  mockWithRemoteTags([]string{"test1"}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.GetLatestTagReference()
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetLatestTagReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				return
			}
			hash, err := c.GetHash(tt.wantTagName)
			if err != nil || !reflect.DeepEqual(got.Hash().String(), hash) {
				t.Errorf("Client.GetLatestTagReference() = %v, want %v", got, tt.wantTagName)
			}
		})
	}
}

func TestClient_ReadFiles(t *testing.T) {
	c := mockInit()
	type args struct {
		paths      []string
		ignoreFile []string
		ignoreDir  []string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name:   "ok_single",
			client: c,
			args: args{
				paths:     []string{"file"},
				ignoreDir: []string{".git"},
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_directory",
			client: c,
			args: args{
				paths:     []string{"."},
				ignoreDir: []string{".git"},
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath):         {0, 0},
				fmt.Sprintf("%s/dir/dir_file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_ignore_file",
			client: c,
			args: args{
				paths:      []string{"."},
				ignoreDir:  []string{".git"},
				ignoreFile: []string{"dir_file"},
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.ReadFiles(tt.args.paths, tt.args.ignoreFile, tt.args.ignoreDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ReadFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ReadFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_AddClientAsSubmodule(t *testing.T) {
	c1 := mockInit()
	c2 := mockGtc()
	c2h, _ := c2.GetHash("main")
	c2t, _ := c2.GetLatestTagReference()
	type args struct {
		name string
		subc Client
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:   "ok",
			client: c1,
			args: args{
				name: "base",
				subc: c2,
			},
			asserts: map[string][]string{
				"submoduleCheck": {fmt.Sprintf(" %s base (%s)", c2h, c2t.Name().Short()), ""},
			},
			wantErr: false,
		},
		{
			name:   "ng_dup",
			client: c1,
			args: args{
				name: "base",
				subc: c2,
			},
			asserts: map[string][]string{
				"submoduleCheck": {fmt.Sprintf(" %s base (%s)", c2h, c2t.Name().Short()), ""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.AddClientAsSubmodule(tt.args.name, tt.args.subc); (err != nil) != tt.wantErr {
				t.Errorf("Client.AddClientAsSubmodule() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}
