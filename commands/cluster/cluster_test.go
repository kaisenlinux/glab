package cluster

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewCmdCluster(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdCluster(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Manage GitLab Agents for Kubernetes and their clusters")
}
