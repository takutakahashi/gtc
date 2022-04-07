package gtc

import (
	"fmt"
	"io/ioutil"
	urlutil "net/url"
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
	CreateBranch bool
	DirPath      string
	OriginURL    string
	Revision     string
	AuthorName   string
	AuthorEmail  string
	Auth         AuthMethod
}

type Client struct {
	opt ClientOpt
	r   *git.Repository
}

type AuthMethod struct {
	AuthMethod    transport.AuthMethod
	username      string
	password      string
	sshPrivateKey []byte
}

type Info struct {
	DirPath      string
	Current      string
	BranchHashes map[string]string
	Status       []string
	Submodules   map[string]Info
	Remote       *Info
}

func GetAuth(username, password, sshKeyPath string) (AuthMethod, error) {
	if username != "" && password != "" {
		auth := &http.BasicAuth{
			Username: username,
			Password: password,
		}
		return AuthMethod{AuthMethod: auth, username: username, password: password}, nil
	}
	if username != "" && sshKeyPath != "" {
		sshKey, err := ioutil.ReadFile(sshKeyPath)
		if err != nil {
			return AuthMethod{}, err
		}
		auth, err := ssh.NewPublicKeys(username, sshKey, "")
		if err != nil {
			return AuthMethod{}, err
		}
		// TODO: selectable later
		auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
		return AuthMethod{AuthMethod: auth, username: username, sshPrivateKey: sshKey}, nil
	}
	return AuthMethod{}, errors.New("no auth method was found")
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
	cloneOpt := &git.CloneOptions{
		URL:               opt.OriginURL,
		ReferenceName:     plumbing.NewBranchReferenceName(opt.Revision),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              opt.Auth.AuthMethod,
	}
	r, err := git.PlainClone(opt.DirPath, false, cloneOpt)
	if err == nil {
		return Client{opt: opt, r: r}, nil
	}
	if err != nil && !opt.CreateBranch {
		return Client{}, errors.Wrap(err, "failed to clone")
	}
	cloneOpt = &git.CloneOptions{
		URL:               opt.OriginURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              opt.Auth.AuthMethod,
	}
	r, err = git.PlainClone(opt.DirPath, false, cloneOpt)
	if err == nil {
		c := Client{opt: opt, r: r}
		if err := c.Checkout(opt.Revision, true); err != nil {
			return Client{}, err
		}
		return c, nil
	}
	return Client{}, errors.Wrap(err, "failed to clone")
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
	if c.r == nil {
		return false
	}
	err := c.r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       c.opt.Auth.AuthMethod,
	})
	return err == nil || err == git.NoErrAlreadyUpToDate
}

func (c *Client) Fetch() error {
	err := c.r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       c.opt.Auth.AuthMethod,
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
	if err := c.r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       c.opt.Auth.AuthMethod,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func (c *Client) Pull(branch string) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	po, err := pullOpt("origin", &c.opt.Auth.AuthMethod)
	if err != nil {
		return err
	}
	po.ReferenceName = plumbing.NewBranchReferenceName(branch)
	if err := w.Pull(po); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func (c *Client) PullAll() error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	po, err := pullOpt("origin", &c.opt.Auth.AuthMethod)
	if err != nil {
		return err
	}
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

func (c *Client) SubmoduleAdd(name, url, revision string, auth *AuthMethod) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	if _, err := w.Submodule(name); err != git.ErrSubmoduleNotFound {
		return err
	}
	repositoryURL := url
	if auth != nil {
		if auth.username != "" && auth.password != "" {
			l, err := urlutil.Parse(url)
			if err != nil {
				return err
			}
			repositoryURL = fmt.Sprintf("%s://%s:%s@%s%s", l.Scheme, auth.username, auth.password, l.Host, l.Path)
		}
	}
	if out, err := c.gitExec([]string{"submodule", "add", "-b", revision, repositoryURL, name}); err != nil {
		return errors.Wrapf(err, "stderr: %s", out)
	}
	return nil
}

func (c *Client) SubmoduleUpdate(remote bool) error {
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	submodules, err := w.Submodules()
	if err != nil {
		return err
	}
	for _, sub := range submodules {
		if err := sub.Update(&git.SubmoduleUpdateOptions{
			Init: true,
			Auth: c.opt.Auth.AuthMethod,
		}); err != nil && err != git.ErrSubmoduleAlreadyInitialized {
			return err
		}
		if err := sub.Update(&git.SubmoduleUpdateOptions{
			Init: false,
			Auth: c.opt.Auth.AuthMethod,
		}); err != nil {
			return err
		}
		sr, err := sub.Repository()
		if err != nil {
			return err
		}
		sw, err := sr.Worktree()
		if err != nil {
			return err
		}
		if err := sw.Pull(&git.PullOptions{
			Auth:  c.opt.Auth.AuthMethod,
			Force: true,
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			return errors.Wrap(err, "failed to pull submodule")
		}
	}
	return nil
}

func (c *Client) SubmoduleSyncUpToDate(message string) error {
	if err := c.SubmoduleUpdate(true); err != nil {
		return err
	}
	w, err := c.r.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	if !status.IsClean() {
		if out, err := c.gitExec([]string{"add", "-A"}); err != nil {
			return errors.Wrapf(err, "failed to add stage. %s", out)
		}
		if err := c.Commit(message); err != nil {
			return err
		}
		if err := c.Push(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) gitExec(commands []string) ([]string, error) {
	if d := os.Getenv("GTC_DEBUG"); d == "true" {
		logrus.Infof("execute command: git %v", commands)
	}
	cmd := exec.Command("git", commands...)
	cmd.Dir = c.opt.DirPath
	b, err := cmd.CombinedOutput()
	return strings.Split(string(b), "\n"), err
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

func (c *Client) CreateBranch(dst string, recreate bool) error {
	if recreate {
		if err := c.r.DeleteBranch(dst); err != nil && err != git.ErrBranchNotFound {
			return err
		}
	}

	return c.Checkout(dst, true)
}

func (c *Client) IsClean() (bool, error) {
	w, err := c.r.Worktree()
	if err != nil {
		return false, err
	}
	status, err := w.Status()
	if err != nil {
		return false, err
	}
	return status.IsClean(), nil
}

func (c *Client) GetRevisionReferenceName(name string) (plumbing.ReferenceName, error) {
	var ref plumbing.ReferenceName = plumbing.NewBranchReferenceName(name)
	if _, err := c.r.ResolveRevision(plumbing.Revision(ref)); err == nil {
		return ref, nil
	}
	ref = plumbing.NewTagReferenceName(name)
	if _, err := c.r.ResolveRevision(plumbing.Revision(ref)); err == nil {
		return ref, nil
	}
	return "", errors.New("no reference name was found")
}

func (c *Client) Info() (Info, error) {
	return info(c.r)
}

func info(r *git.Repository) (Info, error) {

	blank, ret := Info{}, Info{
		Submodules: map[string]Info{},
	}
	currentHash, err := r.ResolveRevision(plumbing.Revision("HEAD"))
	if err != nil {
		return blank, err
	}
	ret.Current = currentHash.String()
	w, err := r.Worktree()
	if err != nil {
		return blank, err
	}
	ret.DirPath = w.Filesystem.Root()
	branches, err := r.Branches()
	if err != nil {
		return blank, err
	}
	branchHashes := map[string]string{}
	branches.ForEach(func(r *plumbing.Reference) error {
		branchHashes[r.Name().Short()] = r.Hash().String()
		return nil
	})
	status, err := w.Status()
	if err != nil {
		return blank, err
	}
	ss, err := w.Submodules()
	if err != nil {
		return blank, err
	}
	for _, s := range ss {
		sr, err := s.Repository()
		if err != nil {
			return blank, err
		}
		si, err := info(sr)
		if err != nil {
			return blank, err
		}
		ret.Submodules[s.Config().Path] = si
	}
	ret.BranchHashes = branchHashes
	ret.Status = strings.Split(status.String(), "\n")
	return ret, nil
}
