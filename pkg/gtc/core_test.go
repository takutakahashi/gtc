package gtc

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
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

func mockOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:      dir,
		CreateBranch: false,
		OriginURL:    "https://github.com/takutakahashi/gtc.git",
		Revision:     "master",
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
	}
}
func mockBranchOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:     dir,
		OriginURL:   "https://github.com/takutakahashi/gtc.git",
		Revision:    "test",
		AuthorName:  "bob",
		AuthorEmail: "bob@mail.com",
	}
}
func mockNoExistsBranchOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:      dir,
		CreateBranch: true,
		OriginURL:    "https://github.com/takutakahashi/gtc.git",
		Revision:     "new-branch",
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
	}
}

func mockOptBasicAuth() ClientOpt {
	o := mockOpt()
	o.Revision = "main"
	o.OriginURL = "https://github.com/takutakahashi/gtc.git"
	auth, _ := GetAuth(os.Getenv("TEST_BASIC_AUTH_USERNAME"), os.Getenv("TEST_BASIC_AUTH_PASSWORD"), "")
	o.Auth = auth
	return o
}
func mockOptSSHAuth() ClientOpt {
	o := mockOpt()
	o.Revision = "main"
	o.OriginURL = "git@github.com:takutakahashi/gtc.git"
	auth, _ := GetAuth("git", "", os.Getenv("TEST_SSH_PRIVATE_KEY_PATH"))
	o.Auth = auth
	return o
}

func mockInit() Client {
	c, _ := Init(mockOpt())
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, "file"), []byte{0, 0}, 0644)
	c.Add("file")
	os.MkdirAll(fmt.Sprintf("%s/dir", c.opt.DirPath), 0755)
	os.WriteFile(fmt.Sprintf("%s/dir/dir_file", c.opt.DirPath), []byte{0, 0}, 0644)
	c.Add("dir/dir_file")
	c.Commit("init")
	return c
}
func mockGtc() Client {
	opt := mockOpt()
	opt.Revision = "main"
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	return c
}

func mockWithTags(tagNames []string) Client {
	c := mockInit()
	for i, name := range tagNames {
		os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, name), []byte{0, 0, 0}, 0644)
		c.Add(name)
		c.commit(name, time.Now().AddDate(0, 0, i))
		c.gitExec([]string{"tag", name})
	}
	return c
}

func mockWithRemoteTags(tagNames []string) Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	for i, name := range tagNames {
		os.WriteFile(fmt.Sprintf("%s/%s", rc.opt.DirPath, name), []byte{0, 0, 0}, 0644)
		rc.Add(name)
		rc.commit(name, time.Now().AddDate(0, 0, i))
		rc.gitExec([]string{"tag", name})
	}
	return c
}

func mockWithRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	return c

}

func mockWithBehindFromRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	os.WriteFile(fmt.Sprintf("%s/%s", rc.opt.DirPath, "file2"), []byte{0, 0}, 0644)
	rc.Add("file2")
	rc.Commit("commit")
	return c

}

func mockWithRemoteAndDirty() Client {
	c := mockWithRemote()
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, "file2"), []byte{0, 0}, 0644)
	c.Add("file2")
	c.Commit("add")
	return c
}

func mockWithSubmodule() Client {
	c1 := mockWithRemote()
	c2 := mockWithRemote()
	c2.AddClientAsSubmodule("test", c1)
	os.WriteFile(fmt.Sprintf("%s/%s", c1.opt.DirPath, "file3"), []byte{0, 0}, 0644)
	c1.Add("file3")
	c1.Commit("add")
	return c2
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
		auth     *transport.AuthMethod
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
	tests := []struct {
		name    string
		client  Client
		asserts map[string][]string
		wantErr bool
	}{
		{
			name:   "ok",
			client: mockWithSubmodule(),
			asserts: map[string][]string{
				"status": {"A  .gitmodules", "A  test", ""},
			},
			wantErr: false,
		},
		{
			name:   "still_ok",
			client: mockInit(),
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
			if err := c.SubmoduleUpdate(); (err != nil) != tt.wantErr {
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
