package gtc

import (
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
		dirPath:   dir,
		originURL: "https://github.com/takutakahashi/gtc.git",
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
