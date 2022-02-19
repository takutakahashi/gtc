package gtc

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ssh2 "golang.org/x/crypto/ssh"
)

type ClientOpt struct {
	DirPath     string
	OriginURL   string
	AuthorName  string
	AuthorEmail string
	Auth        transport.AuthMethod
}

type Client struct {
	opt ClientOpt
	r   *git.Repository
}

func GetAuth(username, password, sshKeyPath string) (transport.AuthMethod, error) {
	if username != "" && password != "" {
		auth := &http.BasicAuth{
			Username: username,
			Password: password,
		}
		return auth, nil
	}
	if username != "" && sshKeyPath != "" {
		sshKey, err := ioutil.ReadFile(sshKeyPath)
		if err != nil {
			return nil, err
		}
		auth, err := ssh.NewPublicKeys(username, sshKey, "")
		if err != nil {
			return nil, err
		}
		// TODO: selectable later
		auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
		return auth, nil
	}
	return nil, errors.New("no auth method was found")
}

func Init(opt ClientOpt) (Client, error) {
	r, err := git.PlainInit(opt.DirPath, false)
	if err != nil {
		return Client{}, err
	}
	return Client{opt: opt, r: r}, nil
}

func Open(opt ClientOpt) (Client, error) {
	r, err := git.PlainOpen(opt.DirPath)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to open")
	}
	return Client{opt: opt, r: r}, nil
}
func Clone(opt ClientOpt) (Client, error) {
	cloneOpt, err := cloneOpt(opt.OriginURL, &opt.Auth)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to clone")
	}
	r, err := git.PlainClone(opt.DirPath, false, cloneOpt)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to clone")
	}
	return Client{opt: opt, r: r}, nil
}

func (c *Client) Add(filePath string) error {
	if c.r == nil {
		return errors.New("this repository is not initialized")
	}
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	_, err = w.Add(filePath)
	return err
}

func (c *Client) Clean() error {
	return os.RemoveAll(c.opt.DirPath)
}

func (c *Client) Initialized() bool {
	if c.r == nil {
		return false
	}
	_, err := c.r.Worktree()
	return err == nil
}

func (c *Client) InitializedWithRemote() bool {
	out, err := c.gitExec([]string{"remote", "show"})
	if err != nil {
		return false
	}
	_, err = c.gitExec([]string{"remote", "show", out[0]})
	return err == nil
}

func (c *Client) Fetch() error {
	err := c.r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       c.opt.Auth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func (c *Client) Commit(message string) error {
	return c.commit(message, time.Now())
}

func (c *Client) commit(message string, date time.Time) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  c.opt.AuthorName,
			Email: c.opt.AuthorEmail,
			When:  date,
		},
	})
	return err
}

func (c *Client) Push() error {
	return c.r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       c.opt.Auth,
	})
}

func (c *Client) Pull(branch string) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	po, err := pullOpt("origin", &c.opt.Auth)
	if err != nil {
		return err
	}
	po.ReferenceName = plumbing.NewBranchReferenceName(branch)
	if err := w.Pull(po); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

// Checkout is the function switchng another refs.
// When force is true, create and switch new branch if named branch is not defined.
func (c *Client) Checkout(name string, force bool) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	return w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
		Create: force,
		Force:  force,
	})
}

func (c *Client) SubmoduleAdd(name, url string, auth *transport.AuthMethod) error {
	if out, err := c.gitExec([]string{"submodule", "add", url, name}); err != nil {
		return errors.Wrapf(err, "stderr: %s", out)
	}
	return nil
}

func (c *Client) SubmoduleUpdate() error {
	if _, err := c.gitExec([]string{"submodule", "init"}); err != nil {
		return err
	}
	if _, err := c.gitExec([]string{"submodule", "update", "--remote"}); err != nil {
		return err
	}
	return nil
}

func (c *Client) SubmoduleSyncUpToDate(message string) error {
	if err := c.SubmoduleUpdate(); err != nil {
		return err
	}
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	s, err := w.Submodules()
	if err != nil {
		return err
	}
	ss, err := s.Status()
	if err != nil {
		return err
	}
	if len(ss) == 0 {
		return errors.New("no submodule found")
	}
	for _, status := range ss {
		if err := c.Add(status.Path); err != nil {
			return err
		}
	}
	if err := c.Commit(message); err != nil {
		return err
	}
	return nil
}

func (c *Client) gitExec(commands []string) ([]string, error) {
	cmd := exec.Command("git", commands...)
	cmd.Dir = c.opt.DirPath
	b, err := cmd.CombinedOutput()
	return strings.Split(string(b), "\n"), err
}

func cloneOpt(url string, auth *transport.AuthMethod) (*git.CloneOptions, error) {
	opt := &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              *auth,
	}
	if opt.Auth == nil {
		logrus.Warn("no authentication parameter was found. no auth method will be used")
	}
	return opt, nil
}

func pullOpt(remoteName string, auth *transport.AuthMethod) (*git.PullOptions, error) {
	opt := &git.PullOptions{
		RemoteName:        remoteName,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              *auth,
	}
	if opt.Auth == nil {
		logrus.Warn("no authentication parameter was found. no auth method will be used")
	}
	return opt, nil
}
