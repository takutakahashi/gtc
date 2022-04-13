package gtc

import (
	"fmt"
	"os"
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
		base        string
		referRemote bool
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
				base:        "master",
				referRemote: false,
			},
			wantErr: false,
		},
		{
			name:   "ok_tag",
			client: mockGtc(),
			args: args{
				base:        "v0.1.0",
				referRemote: false,
			},
			wantErr: false,
		},
		{
			name:   "ok_revision",
			client: mockGtc(),
			args: args{
				base:        "6cac01a031dd3e38ed7fcb12bf6e4e4c08c0b3d7",
				referRemote: false,
			},
			wantErr: false,
		},
		{
			name:   "ng_revision",
			client: mockInit(),
			args: args{
				base:        "6cac01a031dd3e38ed7fcb12bf6e4e4c08c0b3d7",
				referRemote: false,
			},
			wantErr: true,
		},
		{
			name:   "ng_norev",
			client: mockInit(),
			args: args{
				base:        "no-rev",
				referRemote: false,
			},
			wantErr: true,
		},
		{
			name:   "ok_remote",
			client: mockWithRemote(),
			args: args{
				base:        "test",
				referRemote: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.GetHash(tt.args.base, tt.args.referRemote)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				return
			}
			base := tt.args.base
			if tt.args.referRemote {
				base = fmt.Sprintf("remotes/origin/%s", tt.args.base)
			}
			if out, err := c.gitExec([]string{"rev-parse", base}); err != nil || strings.Compare(out[0], got) != 0 {
				t.Errorf("wrong revision. expected: %s, actual: %s", got, out[0])
				return
			}
		})
	}
}

func TestClient_GetLatestTagReference(t *testing.T) {
	type args struct {
		referRemote bool
	}
	tests := []struct {
		name        string
		client      Client
		args        args
		wantTagName string
		want        *plumbing.Reference
		wantErr     bool
	}{
		{
			name:        "ok_multiple_tag",
			client:      mockWithTags([]string{"test1", "test3", "test2"}),
			args:        args{referRemote: false},
			wantTagName: "test2",
			wantErr:     false,
		},
		{
			name:        "ok_single_tag",
			client:      mockWithTags([]string{"test1"}),
			args:        args{referRemote: false},
			wantTagName: "test1",
			wantErr:     false,
		},
		{
			name:    "ng_no_tag",
			client:  mockInit(),
			args:    args{referRemote: false},
			wantErr: true,
		},
		{
			name:    "ng_only_remote_tag",
			client:  mockWithRemoteTags([]string{"test1"}),
			args:    args{referRemote: false},
			wantErr: true,
		},
		{
			name:        "ok_remote_tag",
			client:      mockWithRemoteTags([]string{"test0", "test1"}),
			args:        args{referRemote: true},
			wantTagName: "test1",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.GetLatestTagReference(tt.args.referRemote)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetLatestTagReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				return
			}
			hash, err := c.GetHash(tt.wantTagName, tt.args.referRemote)
			if err != nil || !reflect.DeepEqual(got.Hash().String(), hash) {
				t.Errorf("Client.GetLatestTagReference() = %v, want %v", got, tt.wantTagName)
			}
		})
	}
}

func TestClient_ReadFiles(t *testing.T) {
	c := mockInit()
	type args struct {
		paths        []string
		ignoreFile   []string
		ignoreDir    []string
		absolutePath bool
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
				paths:        []string{"file"},
				ignoreDir:    []string{".git"},
				absolutePath: true,
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_single_noabs",
			client: c,
			args: args{
				paths:        []string{"file"},
				ignoreDir:    []string{".git"},
				absolutePath: false,
			},
			want: map[string][]byte{
				"file": {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_directory",
			client: c,
			args: args{
				paths:        []string{"."},
				ignoreDir:    []string{".git"},
				absolutePath: true,
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath):         {0, 0},
				fmt.Sprintf("%s/dir/dir_file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_directory_noabs",
			client: c,
			args: args{
				paths:        []string{"."},
				ignoreDir:    []string{".git"},
				absolutePath: false,
			},
			want: map[string][]byte{
				"file":         {0, 0},
				"dir/dir_file": {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_ignore_file",
			client: c,
			args: args{
				paths:        []string{"."},
				ignoreDir:    []string{".git"},
				ignoreFile:   []string{"dir_file"},
				absolutePath: true,
			},
			want: map[string][]byte{
				fmt.Sprintf("%s/file", c.opt.DirPath): {0, 0},
			},
			wantErr: false,
		},
		{
			name:   "ok_ignore_file_no_abs",
			client: c,
			args: args{
				paths:        []string{"."},
				ignoreDir:    []string{".git"},
				ignoreFile:   []string{"dir_file"},
				absolutePath: false,
			},
			want: map[string][]byte{
				"file": {0, 0},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.ReadFiles(tt.args.paths, tt.args.ignoreFile, tt.args.ignoreDir, tt.args.absolutePath)
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
				"status": {"A  .gitmodules", "A  base", ""},
			},
			wantErr: false,
		},
		{
			name:   "ok_dup",
			client: c1,
			args: args{
				name: "base",
				subc: c2,
			},
			asserts: map[string][]string{
				"status": {"A  .gitmodules", "A  base", ""},
			},
			wantErr: false,
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

func TestClient_MirrorBranch(t *testing.T) {
	os.Setenv("GTC_DEBUG", "true")
	type args struct {
		src string
		dst string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		wantErr bool
	}{
		{
			name:   "ok_from_local",
			client: mockWithRemote(),
			args: args{
				src: "master",
				dst: "master2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			t.Log(c)
			if err := c.MirrorBranch(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("Client.MirrorBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			srcout, err := c.gitExec([]string{"rev-parse", fmt.Sprintf("remotes/origin/%s", tt.args.src)})
			if err != nil {
				t.Errorf("Client.MirrorBranch() error = %v", err)
			}
			dstout, err := c.gitExec([]string{"rev-parse", fmt.Sprintf("remotes/origin/%s", tt.args.dst)})
			if err != nil {
				t.Errorf("Client.MirrorBranch() error = %v", err)
			}
			if !reflect.DeepEqual(srcout, dstout) {
				t.Errorf("Assertion Error on Client.MirrorBranch(): src: %v, dst: %v", srcout, dstout)
			}
		})
	}
}
