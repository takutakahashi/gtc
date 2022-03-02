package gtc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

func (c *Client) addFile(path string, fileBlob []byte) error {
	filePath := fmt.Sprintf("%s/%s", c.opt.DirPath, path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, fileBlob, 0644)
}

func (c *Client) CommitFiles(files map[string][]byte, message string) error {
	for path, blob := range files {
		if err := c.addFile(path, blob); err != nil {
			return err
		}
		if err := c.Add(path); err != nil {
			return err
		}
	}
	if clean, err := c.IsClean(); err == nil && clean {
		return nil
	} else if err != nil {
		return err
	}

	return c.Commit(message)
}

func (c *Client) GetHash(base string) (string, error) {
	if h, err := c.r.ResolveRevision(plumbing.Revision(plumbing.NewBranchReferenceName(base))); err == nil {
		return h.String(), nil
	}
	if h, err := c.r.ResolveRevision(plumbing.Revision(plumbing.NewTagReferenceName(base))); err == nil {
		return h.String(), nil
	}
	if o, err := c.r.Object(plumbing.CommitObject, plumbing.NewHash(base)); err == nil && !o.ID().IsZero() {
		return base, nil
	}
	return "", errors.New("invalid base reference")
}

func (c *Client) GetLatestTagReference() (*plumbing.Reference, error) {
	tags, err := c.r.Tags()
	if err != nil {
		return nil, err
	}
	latestTagDate := time.Unix(0, 0)
	var latestTagReference *plumbing.Reference = nil
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		commit, err := c.r.CommitObject(ref.Hash())
		if err != nil {
			return err
		}
		if latestTagDate.Before(commit.Author.When) {
			latestTagDate = commit.Author.When
			latestTagReference = ref
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if latestTagReference == nil {
		return nil, errors.New("no tag was found")
	}
	return latestTagReference, nil
}

func (c *Client) ReadFiles(paths, ignoreFile, ignoreDir []string, absolutePath bool) (map[string][]byte, error) {
	result := map[string][]byte{}
	for _, path := range paths {
		buf, err := readFiles(fmt.Sprintf("%s/%s", c.opt.DirPath, path), ignoreFile, ignoreDir)
		if err != nil {
			return nil, err
		}
		for k, v := range buf {
			if absolutePath {
				result[k] = v
			} else {
				result[strings.Replace(k, fmt.Sprintf("%s/", c.opt.DirPath), "", -1)] = v
			}
		}
	}
	return result, nil
}

func readFiles(path string, ignoreFile, ignoreDir []string) (map[string][]byte, error) {
	if ignoreFile == nil {
		ignoreFile = []string{}
	}
	if ignoreDir == nil {
		ignoreDir = []string{}
	}
	ret := map[string][]byte{}
	s, err := os.Stat(path)
	if err != nil {
		return ret, nil
	}
	if !s.IsDir() {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		ret[path] = b
		return ret, nil
	}
	err = filepath.Walk(path, func(path string, info os.FileInfo, e error) error {
		if info.IsDir() {
			for _, s := range ignoreDir {
				if info.Name() == s {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, s := range ignoreFile {
			if strings.Contains(info.Name(), s) {
				return nil
			}
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		ret[path] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) AddClientAsSubmodule(name string, subc Client) error {
	return c.SubmoduleAdd(name, subc.opt.OriginURL, subc.opt.Revision, &subc.opt.Auth)
}

func (c *Client) MirrorBranch(src, dst string) error {
	refs := fmt.Sprintf("remotes/origin/%s:refs/heads/%s", src, dst)
	if _, err := c.gitExec([]string{"push", "origin", refs}); err != nil {
		return err
	}
	// rs := config.RefSpec(refs)
	// if err := c.r.Push(&git.PushOptions{
	// 	RemoteName: "origin",
	// 	Auth:       c.opt.Auth,
	// 	RefSpecs:   []config.RefSpec{rs},
	// }); err != nil && err != git.NoErrAlreadyUpToDate {
	// 	return err
	// }
	return nil
}
