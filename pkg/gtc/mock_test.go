package gtc

import (
	"testing"
)

func TestNewMock(t *testing.T) {
	type args struct {
		o MockOpt
	}
	tests := []struct {
		name    string
		args    args
		asserts map[string][]string
		wantErr bool
	}{
		{
			name: "mock_with_staged_file",
			args: args{
				o: MockOpt{
					CurrentBranch: "master",
					Commits: []MockCommit{
						{
							Message: "initial commit",
						},
					},
					StagedFile: map[string][]byte{
						"file1":     {0, 0},
						"dir/file2": {0, 0},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mock_with_unstaged_file",
			args: args{
				o: MockOpt{
					CurrentBranch: "master",
					Commits: []MockCommit{
						{
							Message: "initial commit",
						},
					},
					UnstagedFile: map[string][]byte{
						"file1":     {0, 0},
						"dir/file2": {0, 0},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mock_with_branch",
			args: args{
				o: MockOpt{
					CurrentBranch: "master",
					Branches:      []string{"master", "master2", "master3"},
					Commits: []MockCommit{
						{
							Message: "initial commit",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMock(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestMock_DirPath(t *testing.T) {
	c := mockInit()
	type fields struct {
		C  Client
		RC *Client
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "ok",
			fields: fields{
				C:  c,
				RC: nil,
			},
			want: c.opt.DirPath,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mock{
				C:  tt.fields.C,
				RC: tt.fields.RC,
			}
			if got := m.DirPath(); got != tt.want {
				t.Errorf("Mock.DirPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
