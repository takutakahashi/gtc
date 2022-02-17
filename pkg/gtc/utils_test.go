package gtc

import (
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestClient_CommitFiles(t *testing.T) {
	type fields struct {
		opt ClientOpt
		r   *git.Repository
	}
	type args struct {
		files   map[string][]byte
		message string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				opt: tt.fields.opt,
				r:   tt.fields.r,
			}
			if err := c.CommitFiles(tt.args.files, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("Client.CommitFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
