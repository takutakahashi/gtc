package gtc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
