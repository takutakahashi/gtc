package gtc

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	ssh2 "golang.org/x/crypto/ssh"
)

type gitCommand []string

var currentBranch gitCommand = []string{"branch", "--show-current"}
var gitStatus gitCommand = []string{"status", "-s"}

func (c *Client) gatherInfo() (map[string][]string, error) {
	result := map[string][]string{}
	ret, err := c.gitExec(currentBranch)
	result["branch"] = ret
	if err != nil {
		return nil, err
	}
	ret, err = c.gitExec(gitStatus)
	if err != nil {
		return nil, err
	}
	result["status"] = ret
	return result, nil
}

func assertion(t *testing.T, c Client, asserts map[string][]string) {
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
		dirPath:     dir,
		originURL:   "https://github.com/takutakahashi/gtc.git",
		authorName:  "bob",
		authorEmail: "bob@mail.com",
	}
}

func mockOptBasicAuth() ClientOpt {
	o := mockOpt()
	o.originURL = "https://github.com/takutakahashi/private-repository-test.git"
	auth := &http.BasicAuth{
		Username: os.Getenv("TEST_BASIC_AUTH_USERNAME"),
		Password: os.Getenv("TEST_BASIC_AUTH_PASSWORD"),
	}
	o.auth = auth
	return o
}
func mockOptSSHAuth() ClientOpt {
	o := mockOpt()
	o.originURL = "git@github.com:takutakahashi/gtc.git"
	sshKey, _ := ioutil.ReadFile(os.Getenv("TEST_SSH_PRIVATE_KEY_PATH"))
	auth, _ := ssh.NewPublicKeys("git", sshKey, "")
	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
	o.auth = auth
	return o
}

func mockInit() Client {
	c, _ := Init(mockOpt())
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.dirPath, "file"), []byte{0, 0}, 0644)
	c.Add("file")
	c.Commit("init")
	return c
}

func mockWithRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.originURL = rc.opt.dirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	return c

}

func mockWithBehindFromRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.originURL = rc.opt.dirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	os.WriteFile(fmt.Sprintf("%s/%s", rc.opt.dirPath, "file2"), []byte{0, 0}, 0644)
	rc.Add("file2")
	rc.Commit("commit")
	return c

}

func mockWithRemoteAndDirty() Client {
	c := mockWithRemote()
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.dirPath, "file2"), []byte{0, 0}, 0644)
	c.Add("file2")
	c.Commit("add")
	return c
}

func mockWithRemoteAndNoCommitedFile() Client {
	c := mockWithRemote()
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.dirPath, "file2"), []byte{0, 0}, 0644)
	return c
}

func TestClone(t *testing.T) {
	type args struct {
		opt ClientOpt
	}
	noAuth := mockOpt()
	basicAuth := mockOptBasicAuth()
	sshAuth := mockOptSSHAuth()
	tests := []struct {
		name    string
		args    args
		want    Client
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
			name: "clone_basic_auth",
			args: args{
				opt: basicAuth,
			},
			wantErr: false,
		},
		{
			name: "clone_ssh_auth",
			args: args{
				opt: sshAuth,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Clone(tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			files, err := os.ReadDir(got.opt.dirPath)
			if err != nil {
				t.Errorf("Clone() error = %v", err)
			}
			isGitRepo := false
			for _, f := range files {
				if f.Name() == ".git" {
					isGitRepo = true
					break
				}
			}
			if !isGitRepo {
				t.Errorf("Clone() failed. this dir is not git repository.")
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
			wantErr: false,
		},
		{
			name:    "up-to-date",
			client:  mockWithRemote(),
			wantErr: true,
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
