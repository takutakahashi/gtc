package gtc

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
)

type gitCommand []string

var currentBranch gitCommand = []string{"branch", "--show-current"}
var listBranch gitCommand = []string{"branch", "-l"}
var gitStatus gitCommand = []string{"status", "-s"}
var gitDiffFile gitCommand = []string{"diff", "--name-only", "HEAD~"}
var latestCommitMessage gitCommand = []string{"log", "-1", "--pretty=%B"}
var submoduleCheck gitCommand = []string{"submodule", "status"}

func (c *Client) gatherInfo() (map[string][]string, error) {
	result := map[string][]string{}
	ret, err := c.gitExec(currentBranch)
	result["branch"] = ret
	if err != nil {
		result["branch"] = nil
	}
	ret, err = c.gitExec(gitStatus)
	if err != nil {
		result["status"] = nil
	}
	result["status"] = ret
	ret, err = c.gitExec(gitDiffFile)
	if err != nil {
		result["diff"] = nil
	}
	result["diff"] = ret

	ret, err = c.gitExec(latestCommitMessage)
	if err != nil {
		result["latestCommitMessage"] = nil
	}
	result["latestCommitMessage"] = ret

	ret, err = c.gitExec(submoduleCheck)
	if err != nil {
		result["submoduleCheck"] = nil
	}
	result["submoduleCheck"] = ret
	ret, err = c.gitExec(listBranch)
	if err != nil {
		result["listBranch"] = nil
	}
	result["listBranch"] = ret
	return result, nil
}

func assertion(t *testing.T, c Client, asserts map[string][]string) {
	t.Log(c)
	info, _ := c.gatherInfo()
	for k, v := range asserts {
		if !reflect.DeepEqual(info[k], v) {
			t.Errorf("assetion failed for %v, expected: %v, actual: %v", k, v, info[k])
		}
	}
}

func TestClone(t *testing.T) {
	type args struct {
		opt ClientOpt
	}
	noAuth := mockOpt()
	noAuth.Revision = "main"
	basicAuth := mockOptBasicAuth()
	// sshAuth := mockOptSSHAuth()
	tests := []struct {
		name    string
		args    args
		want    Client
		asserts map[string][]string
		wantErr bool
	}{
		{
			name: "clone_without_credential",
			args: args{
				opt: noAuth,
			},
			wantErr: false,
		},
		{
			name: "clone_with_branch",
			args: args{
				opt: mockBranchOpt(),
			},
			asserts: map[string][]string{
				"branch": {"test", ""},
			},
			wantErr: false,
		},
		{
			name: "clone_with_create_branch",
			args: args{
				opt: mockNoExistsBranchOpt(),
			},
			asserts: map[string][]string{
				"branch": {"new-branch", ""},
			},
			wantErr: false,
		},
		{
			name: "clone_basic_auth",
			args: args{
				opt: basicAuth,
			},
			wantErr: false,
		},
		// {
		// 	name: "clone_ssh_auth",
		// 	args: args{
		// 		opt: sshAuth,
		// 	},
		// 	wantErr: false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Clone(tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assertion(t, got, tt.asserts)
		})
	}
}

func TestOpen(t *testing.T) {
	c := mockInit()
	type args struct {
		opt ClientOpt
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				opt: ClientOpt{
					DirPath: c.opt.DirPath,
				},
			},
			wantErr: false,
		},
		{
			name: "ng",
			args: args{
				opt: ClientOpt{
					DirPath: "/no_git_dir",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Open(tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_Add(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:    "ok_local",
			client:  mockInit(),
			args:    args{filePath: "file"},
			wantErr: false,
		},
		{
			name:    "no file_local",
			client:  mockInit(),
			args:    args{filePath: "no_file"},
			wantErr: true,
		},
		{
			name:    "ok_remote",
			client:  mockWithRemote(),
			args:    args{filePath: "file"},
			wantErr: false,
		},
		{
			name:    "no file_remote",
			client:  mockWithRemote(),
			args:    args{filePath: "no_file"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			t.Log(c.opt)
			if err := c.Add(tt.args.filePath); (err != nil) != tt.wantErr {
				t.Errorf("Client.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})

	}
}

func TestClient_Commit(t *testing.T) {
	type args struct {
		message string
	}
	tests := []struct {
		name    string
		args    args
		asserts map[string][]string
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				message: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mockInit()
			if err := c.Commit(tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Client.Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_Push(t *testing.T) {
	tests := []struct {
		name    string
		client  Client
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:    "ok",
			client:  mockWithRemoteAndDirty(),
			asserts: map[string][]string{},
			wantErr: false,
		},
		{
			name:    "up-to-date",
			client:  mockWithRemote(),
			wantErr: false,
		},
		{
			name:    "no-remote",
			client:  mockInit(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := tt.client.Push(); (err != nil) != tt.wantErr {
				t.Errorf("Client.Push() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_PullAll(t *testing.T) {
	tests := []struct {
		name    string
		client  Client
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:   "ok",
			client: mockWithBehindFromRemote(),
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: false,
		},
		{
			name:   "up-to-date",
			client: mockWithRemote(),
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: false,
		},
		{
			name:   "NG",
			client: mockInit(),
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.PullAll(); (err != nil) != tt.wantErr {
				t.Errorf("Client.PullAll() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_Pull(t *testing.T) {
	type args struct {
		branch string
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
			client: mockWithBehindFromRemote(),
			args:   args{branch: "master"},
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: false,
		},
		{
			name:   "up-to-date",
			client: mockWithRemote(),
			args:   args{branch: "master"},
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: false,
		},
		{
			name:   "NG",
			client: mockInit(),
			args:   args{branch: "master"},
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: true,
		},
		{
			name:   "NG_with_remote",
			client: mockWithRemote(),
			args:   args{branch: "master2"},
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: true,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.Pull(tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("Client.Pull() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_Checkout(t *testing.T) {
	type args struct {
		name  string
		force bool
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
			client: mockInit(),
			args: args{
				name:  "master",
				force: false,
			},
			asserts: map[string][]string{
				"branch": {"master", ""},
				"status": {""},
			},
			wantErr: false,
		},
		{
			name:   "ok_force",
			client: mockInit(),
			args: args{
				name:  "master2",
				force: true,
			},
			asserts: map[string][]string{
				"branch": {"master2", ""},
			},
			wantErr: false,
		},
		{
			name:   "ng",
			client: mockInit(),
			args: args{
				name:  "master2",
				force: false,
			},
			asserts: map[string][]string{
				"branch": {"master", ""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.Checkout(tt.args.name, tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("Client.Checkout() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_SubmoduleAdd(t *testing.T) {
	type args struct {
		name     string
		url      string
		revision string
		auth     *AuthMethod
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
			client: mockInit(),
			args: args{
				name:     "test",
				url:      "https://github.com/takutakahashi/gtc.git",
				revision: "main",
			},
			asserts: map[string][]string{},
			wantErr: false,
		},
		{
			name:   "ok_already_exists",
			client: mockWithSubmodule(),
			args: args{
				name:     "test",
				url:      "https://github.com/takutakahashi/gtc.git",
				revision: "main",
			},
			asserts: map[string][]string{},
			wantErr: false,
		},
		{
			name:   "ng",
			client: mockInit(),
			args: args{
				name:     "test",
				url:      "https://github.com/takutakahashi/gtc.git",
				revision: "ng-branch",
			},
			asserts: map[string][]string{},
			wantErr: true,
		},
		{
			name:   "ok_auth",
			client: mockInit(),
			args: args{
				name:     "test",
				url:      "https://github.com/takutakahashi/gtc.git",
				revision: "main",
				auth: &AuthMethod{
					username: "takutakahashi",
					password: os.Getenv("TEST_BASIC_AUTH_PASSWORD"),
				},
			},
			asserts: map[string][]string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.SubmoduleAdd(tt.args.name, tt.args.url, tt.args.revision, tt.args.auth); (err != nil) != tt.wantErr {
				t.Errorf("Client.SubmoduleAdd() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_SubmoduleUpdate(t *testing.T) {
	type args struct {
		remote bool
	}
	tests := []struct {
		name    string
		args    args
		client  Client
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:   "ok",
			client: mockWithSubmodule(),
			args: args{
				remote: false,
			},
			asserts: map[string][]string{
				"status": {"A  .gitmodules", "A  test", ""},
			},
			wantErr: false,
		},
		{
			name:   "ok_remote",
			client: mockWithSubmodule(),
			args: args{
				remote: true,
			},
			asserts: map[string][]string{
				"status": {"A  .gitmodules", "A  test", ""},
			},
			wantErr: false,
		},
		{
			name:   "still_ok",
			client: mockInit(),
			args: args{
				remote: false,
			},
			asserts: map[string][]string{
				"status": {""},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			t.Log(c)
			if err := c.SubmoduleUpdate(tt.args.remote); (err != nil) != tt.wantErr {
				t.Errorf("Client.SubmoduleUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_Clean(t *testing.T) {
	tests := []struct {
		name    string
		client  Client
		wantErr bool
	}{
		{
			name:    "ok",
			client:  mockInit(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if _, err := os.ReadDir(c.opt.DirPath); err != nil {
				t.Errorf("directory is not found. err: %v", err)
			}
			if err := c.Clean(); (err != nil) != tt.wantErr {
				t.Errorf("Client.Clean() error = %v, wantErr %v", err, tt.wantErr)
			}
			if dir, err := os.ReadDir(c.opt.DirPath); err == nil {
				t.Errorf("directory is not deleted. dir: %v", dir)
			}
		})
	}
}

func TestClient_InitializedWithRemote(t *testing.T) {
	tests := []struct {
		name   string
		client Client
		want   bool
	}{
		{
			name:   "ok",
			client: mockWithRemote(),
			want:   true,
		},
		{
			name:   "ng",
			client: mockInit(),
			want:   false,
		},
		{
			name: "ng_no_git_dir",
			client: Client{
				opt: ClientOpt{
					DirPath: "/no_git_dir",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if got := c.InitializedWithRemote(); got != tt.want {
				t.Errorf("Client.InitializedWithRemote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Initialized(t *testing.T) {
	tests := []struct {
		name   string
		client Client
		want   bool
	}{
		{
			name:   "ok",
			client: mockInit(),
			want:   true,
		},
		{
			name: "ng_no_git_dir",
			client: Client{
				opt: ClientOpt{
					DirPath: "/no_git_dir",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if got := c.Initialized(); got != tt.want {
				t.Errorf("Client.Initialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Fetch(t *testing.T) {
	tests := []struct {
		name    string
		client  Client
		wantErr bool
	}{
		{
			name:    "ok",
			client:  mockWithRemote(),
			wantErr: false,
		},
		{
			name:    "ok_behind",
			client:  mockWithBehindFromRemote(),
			wantErr: false,
		},
		{
			name:    "NG",
			client:  mockInit(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.Fetch(); (err != nil) != tt.wantErr {
				t.Errorf("Client.Fetch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_SubmoduleSyncUpToDate(t *testing.T) {
	type args struct {
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
			name:   "ok",
			client: mockWithSubmodule(),
			args: args{
				message: "submodule update",
			},
			asserts: map[string][]string{
				"latestCommitMessage": {"submodule update", ""},
				"status":              {""},
			},
			wantErr: false,
		},
		{
			name:   "ok_clean",
			client: mockWithRemote(),
			args: args{
				message: "submodule update",
			},
			asserts: map[string][]string{
				"latestCommitMessage": {"init", ""},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.SubmoduleSyncUpToDate(tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Client.SubmoduleSyncUpToDate() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_CreateBranch(t *testing.T) {
	type args struct {
		dst      string
		recreate bool
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
			client: mockInit(),
			args: args{
				dst:      "newbranch",
				recreate: true,
			},
			asserts: map[string][]string{
				"listBranch": {"  master", "* newbranch", ""},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			if err := c.CreateBranch(tt.args.dst, tt.args.recreate); (err != nil) != tt.wantErr {
				t.Errorf("Client.CreateBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertion(t, c, tt.asserts)
		})
	}
}

func TestClient_IsClean(t *testing.T) {
	tests := []struct {
		name    string
		client  Client
		want    bool
		wantErr bool
	}{
		{
			name:    "clean",
			client:  mockInit(),
			want:    true,
			wantErr: false,
		},
		{
			name:    "duty",
			client:  mockWithUnstagedFile(),
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.IsClean()
			t.Log(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.IsClean() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.IsClean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetRevisionReferenceName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		client  Client
		args    args
		want    plumbing.ReferenceName
		wantErr bool
	}{
		{
			name:   "ok_tag",
			client: mockWithTags([]string{"v0.1"}),
			args: args{
				name: "v0.1",
			},
			want:    plumbing.NewTagReferenceName("v0.1"),
			wantErr: false,
		},
		{
			name:   "ok_branch",
			client: mockInit(),
			args: args{
				name: "master",
			},
			want:    plumbing.NewBranchReferenceName("master"),
			wantErr: false,
		},
		{
			name:   "ng",
			client: mockInit(),
			args: args{
				name: "v0.1",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.GetRevisionReferenceName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetRevisionReferenceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetRevisionReferenceName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Info(t *testing.T) {

	c := mockWithSubmodule()
	p := c.opt.DirPath
	hash, _ := c.GetHash("master", false)
	w, _ := c.r.Worktree()
	s, _ := w.Submodule("test")
	sr, _ := s.Repository()
	h, _ := sr.ResolveRevision(plumbing.Revision(plumbing.NewBranchReferenceName("master")))
	tests := []struct {
		name    string
		client  Client
		want    Info
		wantErr bool
	}{
		{
			name:   "ok",
			client: c,
			want: Info{
				DirPath: p,
				Current: hash,
				BranchHashes: map[string]string{
					"master": hash,
				},
				Status: []string{
					"A  test",
					"A  .gitmodules",
					"",
				},
				Submodules: map[string]Info{
					"test": {
						Current: h.String(),
						DirPath: fmt.Sprintf("%s/test", p),
						BranchHashes: map[string]string{
							"master": h.String(),
						},
						Submodules: map[string]Info{},
						Status:     []string{""},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.client
			got, err := c.Info()
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Info() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Client.Info() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
