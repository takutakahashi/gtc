package gtc

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type Mock struct {
	// Local Client
	C Client
	// Rmote Repository Client
	RC *Client
}

type MockCommit struct {
	Message string
	Files   map[string][]byte
}

type MockOpt struct {
	OriginURL     string
	CurrentBranch string
	Branches      []string
	Commits       []MockCommit
	StagedFile    map[string][]byte
	UnstagedFile  map[string][]byte
	Remote        *MockOpt
	RC            *Client
}

type SubmoduleOpt struct {
	Path string
}

func NewMock(o MockOpt) (Mock, error) {
	mock := Mock{}
	if o.Remote != nil {
		rm, err := NewMock(*o.Remote)
		if err != nil {
			return Mock{}, err
		}
		mock.RC = &rm.C
	}
	if o.RC != nil {
		mock.RC = o.RC
	}
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	if mock.RC != nil {
		opt := ClientOpt{
			DirPath:      dir,
			OriginURL:    mock.RC.opt.DirPath,
			CreateBranch: false,
			Revision:     o.CurrentBranch,
			AuthorName:   "bob",
			AuthorEmail:  "bob@mail.com",
		}
		c, err := Clone(opt)
		if err != nil {
			return Mock{}, err
		}
		mock.C = c
	} else {
		opt := ClientOpt{
			DirPath:      dir,
			OriginURL:    "",
			CreateBranch: false,
			Revision:     o.CurrentBranch,
			AuthorName:   "bob",
			AuthorEmail:  "bob@mail.com",
		}
		c, err := Init(opt)
		if err != nil {
			return Mock{}, err
		}
		mock.C = c

	}
	if err := mock.compose(o); err != nil {
		return Mock{}, err
	}
	return mock, nil
}

func (m *Mock) compose(o MockOpt) error {
	if m.RC != nil {
		if err := m.C.Pull(o.CurrentBranch); err != nil {
			return err
		}
	}
	// create commit
	for _, commit := range o.Commits {
		for name, blob := range commit.Files {
			os.MkdirAll(filepath.Dir(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name)), 0755)
			if err := os.WriteFile(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name), blob, 0644); err != nil {
				return err
			}
			if err := m.C.Add(name); err != nil {
				return err
			}
		}
		if err := m.C.Commit(commit.Message); err != nil {
			return err
		}
	}
	for _, b := range o.Branches {
		m.C.CreateBranch(b, false)
		m.C.Checkout(o.CurrentBranch, false)
	}
	// create staged file
	for name, blob := range o.StagedFile {
		os.MkdirAll(filepath.Dir(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name)), 0755)
		if err := os.WriteFile(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name), blob, 0644); err != nil {
			return err
		}
		if err := m.C.Add(name); err != nil {
			return err
		}
	}

	// create unstaged file
	for name, blob := range o.UnstagedFile {
		os.MkdirAll(filepath.Dir(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name)), 0755)
		if err := os.WriteFile(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name), blob, 0644); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mock) DirPath() string {
	return m.C.opt.DirPath
}

func (m *Mock) ClientOpt() ClientOpt {
	return m.C.opt
}

func (m *Mock) RemoteClientOpt() ClientOpt {
	return m.RC.opt
}

func (m *Mock) RandomCommitLocal(branch string, push bool) error {
	c := m.C
	if err := c.Checkout(branch, true); err != nil {
		return err
	}
	filename := mkTestName()
	if err := c.CommitFiles(map[string][]byte{
		filename: []byte(filename),
	}, filename); err != nil {
		return err
	}
	return nil
}

func (m *Mock) RandomCommitRemote(branch string) error {
	c := m.RC
	if err := c.Checkout(branch, true); err != nil {
		return err
	}
	filename := mkTestName()
	if err := c.CommitFiles(map[string][]byte{
		filename: []byte(filename),
	}, filename); err != nil {
		return err
	}
	return nil
}

func mkTestName() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic("error")
	}

	var result string
	for _, v := range b {
		result += string(letters[int(v)%len(letters)])
	}
	return fmt.Sprintf("unittest-%s", result)
}

func mockOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:      dir,
		CreateBranch: false,
		OriginURL:    "https://github.com/takutakahashi/gtc.git",
		Revision:     "master",
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
	}
}
func mockBranchOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:     dir,
		OriginURL:   "https://github.com/takutakahashi/gtc.git",
		Revision:    "test",
		AuthorName:  "bob",
		AuthorEmail: "bob@mail.com",
	}
}
func mockNoExistsBranchOpt() ClientOpt {
	dir, _ := ioutil.TempDir("/tmp", "gtc-")
	return ClientOpt{
		DirPath:      dir,
		CreateBranch: true,
		OriginURL:    "https://github.com/takutakahashi/gtc.git",
		Revision:     "new-branch",
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
	}
}

func mockOptBasicAuth() ClientOpt {
	o := mockOpt()
	o.Revision = "main"
	o.OriginURL = "https://github.com/takutakahashi/gtc.git"
	auth, _ := GetAuth(os.Getenv("TEST_BASIC_AUTH_USERNAME"), os.Getenv("TEST_BASIC_AUTH_PASSWORD"), "")
	o.Auth = auth
	return o
}
func mockOptSSHAuth() ClientOpt {
	o := mockOpt()
	o.Revision = "main"
	o.OriginURL = "git@github.com:takutakahashi/gtc.git"
	auth, _ := GetAuth("git", "", os.Getenv("TEST_SSH_PRIVATE_KEY_PATH"))
	o.Auth = auth
	return o
}

func mockInit() Client {
	m, err := NewMock(MockOpt{
		CurrentBranch: "master",
		Commits: []MockCommit{
			{
				Message: "init",
				Files: map[string][]byte{
					"file":         {0, 0},
					"dir/dir_file": {0, 0},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return m.C
}
func mockWithRemote() Client {
	m, err := NewMock(MockOpt{
		Remote: &MockOpt{
			Branches: []string{
				"master", "test",
			},
			CurrentBranch: "master",
			Commits: []MockCommit{
				{
					Message: "init",
					Files: map[string][]byte{
						"file":         {0, 0},
						"dir/dir_file": {0, 0},
					},
				},
			},
		},
		CurrentBranch: "master",
	})
	if err != nil {
		panic(err)
	}
	return m.C
}
func mockWithUnstagedFile() Client {
	m, err := NewMock(MockOpt{
		CurrentBranch: "master",
		UnstagedFile: map[string][]byte{
			"file_mockwithunstagedfile": {0, 0},
		},
	})
	if err != nil {
		panic(err)
	}
	return m.C
}
func mockGtc() Client {
	opt := mockOpt()
	opt.Revision = "main"
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	return c
}

func mockWithTags(tagNames []string) Client {
	c := mockInit()
	for i, name := range tagNames {
		os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, name), []byte{0, 0, 0}, 0644)
		c.Add(name)
		c.commit(name, time.Now().AddDate(0, 0, i))
		c.gitExec([]string{"tag", name})
	}
	return c
}

func mockWithRemoteTags(tagNames []string) Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	for i, name := range tagNames {
		os.WriteFile(fmt.Sprintf("%s/%s", rc.opt.DirPath, name), []byte{0, 0, 0}, 0644)
		rc.Add(name)
		rc.commit(name, time.Now().AddDate(0, 0, i))
		rc.gitExec([]string{"tag", name})
	}
	return c
}

func mockWithBehindFromRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
	}
	os.WriteFile(fmt.Sprintf("%s/%s", rc.opt.DirPath, "file2"), []byte{0, 0}, 0644)
	rc.Add("file2")
	rc.Commit("commit")
	return c

}

func mockWithRemoteAndDirty() Client {
	c := mockWithRemote()
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, "file2"), []byte{0, 0}, 0644)
	c.Add("file2")
	c.Commit("add")
	return c
}

func mockWithSubmodule() Client {
	c1 := mockWithRemote()
	m2, _ := NewMock(MockOpt{
		CurrentBranch: "master",
		Commits: []MockCommit{
			{
				Message: "test2",
				Files: map[string][]byte{
					"test2": {0, 1, 2, 3, 4},
				},
			},
		},
		Remote: &MockOpt{
			CurrentBranch: "master",
			Commits: []MockCommit{
				{
					Message: "test2",
					Files: map[string][]byte{
						"test2": {0, 1, 2, 3, 4},
					},
				},
			},
		},
	})
	c2 := m2.C
	c2.AddClientAsSubmodule("test", c1)
	return c2
}
