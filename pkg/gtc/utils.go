package gtc

import (
	"fmt"
	"io/ioutil"
)

func (c *Client) addFile(path string, fileBlob []byte) error {
	return ioutil.WriteFile(fmt.Sprintf("%s/%s", c.opt.dirPath, path), fileBlob, 0644)
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
