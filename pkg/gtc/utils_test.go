package gtc

import (
	"strings"
	"testing"
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
