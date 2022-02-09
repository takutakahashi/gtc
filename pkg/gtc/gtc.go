package gtc

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ClientOpt struct {
	dirPath   string
	originURL string
	auth      transport.AuthMethod
}

type Client struct {
	opt ClientOpt
	r   *git.Repository
}

func Clone(opt ClientOpt) (Client, error) {
	cloneOpt, err := CloneOpt(opt.originURL, &opt.auth)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to clone")
	}
	r, err := git.PlainClone(opt.dirPath, false, cloneOpt)
	if err != nil {
		return Client{}, errors.Wrap(err, "failed to clone")
	}
	return Client{opt: opt, r: r}, nil
}

//func (*Client) SubmoduleUpdate() error                         {}

//func (c *Client) Add(filePath string) error                                    {}
//func (c *Client) Status() ([]string, error)                                    {}
//func (c *Client) Commit(message string) error                                  {}
//func (c *Client) Push() error                                                  {}
//func (c *Client) SubmoduleInit(localPath string, url string, auth *Auth) error {}
// func (c *Client) SubmoduleAdd(localPath, url, branch string, auth *Auth) error {
// 	gitmodTemplate := `[submodule "{{.Name}}"]
//     path = {{.Path}}
//     url = {{.URL}}
//     branch = {{.Branch}}
//     `
// 	v := struct {
// 		Name   string
// 		Path   string
// 		URL    string
// 		Branch string
// 	}{
// 		Name:   localPath,
// 		Path:   localPath,
// 		URL:    url,
// 		Branch: branch,
// 	}
// 	t := template.Must(template.New("gitmodule").Parse(gitmodTemplate))
// 	var buf bytes.Buffer
// 	if err := t.Execute(&buf, v); err != nil {
// 		return err
// 	}
// 	if err := ioutil.WriteFile(fmt.Sprintf("%s/.gitmodules", c.opt.dirPath), buf.Bytes(), 0644); err != nil {
// 		return err
// 	}
// 	cmd := exec.Command("git", "submodule", "add", url, localPath)
// 	_, err := cmd.Output()
// 	return errors.Wrap(err, "failed to add submodule")
// }

func CloneOpt(url string, auth *transport.AuthMethod) (*git.CloneOptions, error) {
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

func PullOpt(remoteName string, auth *transport.AuthMethod) (*git.PullOptions, error) {
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

func FetchOpt(remoteName string, auth *transport.AuthMethod) (*git.FetchOptions, error) {
	opt := &git.FetchOptions{
		RemoteName: remoteName,
		Auth:       *auth,
	}
	if opt.Auth == nil {
		logrus.Warn("no authentication parameter was found. no auth method will be used")
	}
	return opt, nil
}
