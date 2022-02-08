package gtc

type Client struct {
  workDir string
  repositoryName string
  originURL string
  auth Auth
}

type Auth struct {}

func (*Client) Clone(url string, auth *Auth) (Client, error) {}
func (*Client) Add(filePath string) error {}
func (*Client) Status() ([]string, error) {}
func (*Client) Commit(message string) error {}
func (*Client) Push() error {}
func (*Client) SubmoduleInit(localPath string, url string, auth *Auth) error {}
func (*Client) SubmoduleUpdate() error {}
