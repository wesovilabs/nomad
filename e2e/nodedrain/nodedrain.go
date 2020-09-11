package nodedrain

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	e2e "github.com/hashicorp/nomad/e2e/e2eutil"
	"github.com/hashicorp/nomad/e2e/framework"
	"github.com/hashicorp/nomad/helper/uuid"
	"github.com/hashicorp/nomad/testutil"
)

type NodeDrainE2ETest struct {
	framework.TC
	jobIDs  []string
	nodeIDs []string
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
	e2e.WaitForNodesReady(f.T(), tc.Nomad(), 2) // needs at least 2 to test migration
}

func (tc *NodeDrainE2ETest) AfterEach(f *framework.F) {
	if os.Getenv("NOMAD_TEST_SKIPCLEANUP") == "1" {
		return
	}

	for _, id := range tc.jobIDs {
		_, err := e2e.Command("nomad", "job", "stop", "-purge", id)
		f.NoError(err)
	}
	tc.jobIDs = []string{}

	for _, id := range tc.nodeIDs {
		_, err := e2e.Command("nomad", "node", "drain", "-disable", id)
		f.NoError(err)
	}
	tc.nodeIDs = []string{}

	_, err := e2e.Command("nomad", "system", "gc")
	f.NoError(err)
}

// nodeDrainEnable starts draining a node
func nodeDrainEnable(nodeID string) error {
	out, err := e2e.Command("nomad", "node", "drain", "-enable", "-detach", nodeID)
	if err != nil {
		return fmt.Errorf("'nomad node drain' failed: %w\n%v", err, out)
	}
	return nil
}

func nodesForJob(jobID string) ([]string, error) {
	allocs, err := e2e.AllocsForJob(jobID)
	if err != nil {
		return nil, err
	}
	nodes := []string{}
	for _, alloc := range allocs {
		nodes = append(nodes, alloc["NodeID"])
	}
	return nodes, nil
}

func nodeForAlloc(allocID string) (string, error) {
	out, err := e2e.Command("nomad", "alloc", "status", "-verbose", allocID)
	if err != nil {
		return "", fmt.Errorf("'nomad alloc status' failed: %w\n%v", err, out)
	}

	nodeID, err := e2e.GetField(out, "Node ID")
	if err != nil {
		return "", fmt.Errorf("could not find Node ID field: %w\n%v", err, out)
	}
	return nodeID, nil
}

// waitForNodeDrainStatus is a convenience wrapper that polls 'node status'
// until the comparison function over the state of the nodes returns true.
func waitForNodeDrainStatus(nodeID string, comparison func([]map[string]string) bool, wc *e2e.WaitConfig) error {
	var got []map[string]string
	var err error
	interval, retries := wc.OrDefault()
	testutil.WaitForResultRetries(retries, func() (bool, error) {
		time.Sleep(interval)
		got, err = e2e.AllocsForNode(nodeID)
		if err != nil {
			return false, err
		}
		return comparison(got), nil
	}, func(e error) {
		err = fmt.Errorf("node drain status check failed: %w\n%#v", e, got)
	})
	return err
}

// TestNodeDrainEphemeralMigrate tests that ephermeral_disk migrations work as
// expected even during a node drain.
func (tc *NodeDrainE2ETest) TestNodeDrainEphemeralMigrate(f *framework.F) {
	jobID := "test-node-drain-" + uuid.Generate()[0:8]
	f.NoError(e2e.Register(jobID, "nodedrain/input/drain_migrate.nomad"))
	tc.jobIDs = append(tc.jobIDs, jobID)

	expected := []string{"running"}
	f.NoError(e2e.WaitForAllocStatusExpected(jobID, expected), "job should be running")

	nodes, err := nodesForJob(jobID)
	f.NoError(err, "could not get nodes for job")
	f.Len(nodes, 1, "could not get nodes for job")
	nodeID := nodes[0]

	var oldAllocID string

	nodeDrainEnable(nodeID)
	f.NoError(waitForNodeDrainStatus(nodeID,
		func(got []map[string]string) bool {
			for _, alloc := range got {
				if alloc["Status"] != "complete" {
					return false
				}
				oldAllocID = alloc["ID"] // we'll want this later
			}
			return true
		}, nil),
		"node did not drain")

	// wait for the allocation to be migrated
	expected = []string{"complete", "running"}
	f.NoError(e2e.WaitForAllocStatusExpected(jobID, expected), "job should be running")

	allocs, err := e2e.AllocsForJob(jobID)
	f.NoError(err, "could not get allocations for job")

	// the job writes its alloc ID to a file if it hasn't been previously
	// written, so find the contents of the migrated file and make sure they
	// match the old allocation, not the running one
	var got string
	var fsErr error
	testutil.WaitForResultRetries(500, func() (bool, error) {
		time.Sleep(time.Millisecond * 100)
		for _, alloc := range allocs {
			if alloc["Status"] == "running" && alloc["Node ID"] != nodeID && alloc["ID"] != oldAllocID {
				got, fsErr = e2e.Command("nomad", "alloc", "fs",
					alloc["ID"], fmt.Sprintf("alloc/data/%s", jobID))
				if err != nil {
					return false, err
				}
				if strings.TrimSpace(got) == oldAllocID {
					return true, nil
				} else {
					return false, fmt.Errorf("expected %q, got %q", oldAllocID, got)
				}
			}
		}
		return false, fmt.Errorf("did not find a migrated alloc")
	}, func(e error) {
		fsErr = e
	})
	f.NoError(fsErr, "node drained but migration failed")
}

// TestNodeDrainIgnoreSystem tests that system jobs are left behind when the
// -ignore-system flag is used.
func (tc *NodeDrainE2ETest) TestNodeDrainIgnoreSystem(f *framework.F) {}

// TestNodeDrainKeepIneligible tests that nodes can be kept ineligible for
// scheduling after disabling drain.
func (tc *NodeDrainE2ETest) TestNodeDrainKeepIneligible(f *framework.F) {}

// TestNodeDrainDeadline tests the enforcement of the node drain deadline so
// that allocations are terminated even if they haven't gracefully exited.
func (tc *NodeDrainE2ETest) TestNodeDrainDeadline(f *framework.F) {}

// TestNodeDrainDeadline tests the enforcement of the node drain -force flag
// so that allocations are terminated immediately.
func (tc *NodeDrainE2ETest) TestNodeDrainForce(f *framework.F) {}

// TestSimple runs a job that should fail and never reschedule
func (tc *NodeDrainE2ETest) TestSimple(f *framework.F) {
	jobID := "test-simple-" + uuid.Generate()[0:8]
	f.NoError(e2e.Register(jobID, "nodedrain/input/example.nomad"))
	tc.jobIDs = append(tc.jobIDs, jobID)

	expected := []string{"running"}
	err := e2e.WaitForAllocStatusExpected(jobID, expected)
	f.NoError(err, "should have exactly 1 running alloc")

	err = e2e.WaitForAllocStatusComparison(
		func() ([]string, error) { return e2e.AllocStatuses(jobID) },
		func(got []string) bool { return reflect.DeepEqual(got, expected) },
		nil,
	)
	f.NoError(err, "should have exactly 1 running alloc")
}
