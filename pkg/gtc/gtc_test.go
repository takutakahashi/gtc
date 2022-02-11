package gtc

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	ssh2 "golang.org/x/crypto/ssh"
)

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
	sshKey, _ := ioutil.ReadFile("/Users/takutaka/.ssh/id_rsa")
	auth, _ := ssh.NewPublicKeys("git", sshKey, "")
	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
	o.auth = auth
	return o
}

func mockGen() (ClientOpt, ClientOpt, ClientOpt) {
	return mockOpt(), mockOptBasicAuth(), mockOptSSHAuth()
}

func mockInit() Client {
	c, _ := Init(mockOpt())
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.dirPath, "file"), []byte{0, 0}, 0644)
	return c
}

func TestClone(t *testing.T) {
	type args struct {
		opt ClientOpt
	}
	noAuth := mockOpt()
	basicAuth := mockOptBasicAuth()
	_ = basicAuth
	sshAuth := mockOptSSHAuth()
	_ = sshAuth
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
		args    args
		wantErr bool
	}{
		{
			name:    "ok",
			args:    args{filePath: "file"},
			wantErr: false,
		},
		{
			name:    "no file",
			args:    args{filePath: "no_file"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mockInit()
			t.Log(c.opt.dirPath)
			if err := c.Add(tt.args.filePath); (err != nil) != tt.wantErr {
				t.Errorf("Client.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
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
			t.Log(c.opt.dirPath)
			if err := c.Commit(tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Client.Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
