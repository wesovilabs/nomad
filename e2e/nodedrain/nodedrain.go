package nodedrain

import (
	"os"
	"reflect"

	e2e "github.com/hashicorp/nomad/e2e/e2eutil"
	"github.com/hashicorp/nomad/e2e/framework"
	"github.com/hashicorp/nomad/helper/uuid"
)

type NodeDrainE2ETest struct {
	framework.TC
	jobIds []string
}

func init() {
	framework.AddSuites(&framework.TestSuite{
		Component:   "NodeDrain",
		CanRunLocal: true,
		Consul:      true,
		Cases: []framework.TestCase{
			new(NodeDrainE2ETest),
		},
	})

}

func (tc *NodeDrainE2ETest) BeforeAll(f *framework.F) {
	e2e.WaitForLeader(f.T(), tc.Nomad())
	e2e.WaitForNodesReady(f.T(), tc.Nomad(), 1)
}

func (tc *NodeDrainE2ETest) AfterEach(f *framework.F) {
	if os.Getenv("NOMAD_TEST_SKIPCLEANUP") == "1" {
		return
	}

	for _, id := range tc.jobIds {
		_, err := e2e.Command("nomad", "job", "stop", "-purge", id)
		f.NoError(err)
	}
	tc.jobIds = []string{}
	_, err := e2e.Command("nomad", "system", "gc")
	f.NoError(err)
}

// TestSimple runs a job that should fail and never reschedule
func (tc *NodeDrainE2ETest) TestSimple(f *framework.F) {
	jobID := "test-simple" + uuid.Generate()[0:8]
	f.NoError(e2e.Register(jobID, "nodedrain/input/example.nomad"))
	tc.jobIds = append(tc.jobIds, jobID)

	expected := []string{"running"}
	err := e2e.WaitForAllocStatusExpected(jobID, expected)
	f.NoError(err, "should have exactly 1 running alloc")

	err = e2e.WaitForAllocStatusComparison(
		func() ([]string, error) { return e2e.AllocStatuses(jobID) },
		func(got []string) bool { return reflect.DeepEqual(got, expected) },
	)
	f.NoError(err, "should have exactly 1 running alloc")
}
