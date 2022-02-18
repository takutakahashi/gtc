package gtc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing"
)

func (c *Client) addFile(path string, fileBlob []byte) error {
	filePath := fmt.Sprintf("%s/%s", c.opt.dirPath, path)
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
