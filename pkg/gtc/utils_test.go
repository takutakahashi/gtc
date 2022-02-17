package gtc

import (
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
