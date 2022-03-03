package gtc

import (
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
	DirPath       string
	OriginURL     string
	CurrentBranch string
	Branches      []string
	Commits       []MockCommit
	StagedFile    map[string][]byte
	UnstagedFile  map[string][]byte
	Remote        *MockOpt
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
	for _, b := range o.Branches {
		if err := m.C.Checkout(b, true); err != nil {
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
	for name, blob := range o.StagedFile {
		os.MkdirAll(filepath.Dir(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name)), 0755)
		if err := os.WriteFile(fmt.Sprintf("%s/%s", m.C.opt.DirPath, name), blob, 0644); err != nil {
			return err
		}
	}
	return nil
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

func newMockInitOpt() MockOpt {
	return MockOpt{
		CurrentBranch: "master",
	}
}

func mockInit() Client {
	m, err := NewMock(MockOpt{
		CurrentBranch: "master",
		Commits: []MockCommit{
			{
				Message: "init",
				Files: map[string][]byte{
					"file":         []byte{0, 0},
					"dir/dir_file": []byte{0, 0},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return m.C
	c, _ := Init(mockOpt())
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, "file"), []byte{0, 0}, 0644)
	c.Add("file")
	os.MkdirAll(fmt.Sprintf("%s/dir", c.opt.DirPath), 0755)
	os.WriteFile(fmt.Sprintf("%s/dir/dir_file", c.opt.DirPath), []byte{0, 0}, 0644)
	c.Add("dir/dir_file")
	c.Commit("init")
	return c
}
func mockWithUnstagedFile() Client {
	c := mockInit()
	os.WriteFile(fmt.Sprintf("%s/%s", c.opt.DirPath, "file_mockwithunstagedfile"), []byte{0, 0}, 0644)
	return c
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

func mockWithRemote() Client {
	rc := mockInit()
	opt := mockOpt()
	opt.OriginURL = rc.opt.DirPath
	c, err := Clone(opt)
	if err != nil {
		panic(err)
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
	c2 := mockWithRemote()
	c2.AddClientAsSubmodule("test", c1)
	os.WriteFile(fmt.Sprintf("%s/%s", c1.opt.DirPath, "file3"), []byte{0, 0}, 0644)
	c1.Add("file3")
	c1.Commit("add")
	return c2
}
