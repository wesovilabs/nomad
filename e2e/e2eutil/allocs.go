package e2eutil

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/testutil"
)

// WaitForAllocStatusExpected polls 'nomad job status' and exactly compares
// the status of all allocations (including any previous versions) against the
// expected list.
func WaitForAllocStatusExpected(jobID string, expected []string) error {
	return WaitForAllocStatusComparison(
		func() ([]string, error) { return AllocStatuses(jobID) },
		func(got []string) bool { return reflect.DeepEqual(got, expected) },
	)
}

// WaitForAllocStatusComparison is a convenience wrapper that polls the query
// function until the comparison function returns true.
func WaitForAllocStatusComparison(query func() ([]string, error), comparison func([]string) bool) error {
	var got []string
	var err error
	testutil.WaitForResultRetries(30, func() (bool, error) {
		time.Sleep(time.Millisecond * 100)
		got, err = query()
		if err != nil {
			return false, err
		}
		return comparison(got), nil
	}, func(e error) {
		err = fmt.Errorf("alloc status check failed: got %#v", got)
	})
	return err
}

// AllocStatuses returns a slice of client statuses
func AllocStatuses(jobID string) ([]string, error) {

	out, err := Command("nomad", "job", "status", "-verbose", "-all-allocs", jobID)
	if err != nil {
		return nil, fmt.Errorf("nomad job status failed: %w", err)
	}

	section, err := GetSection(out, "Allocations")
	if err != nil {
		return nil, fmt.Errorf("could not find Allocations section: %w", err)
	}

	allocs, err := ParseColumns(section)
	if err != nil {
		return nil, fmt.Errorf("could not parse Allocations section: %w", err)
	}

	statuses := []string{}
	for _, alloc := range allocs {
		statuses = append(statuses, alloc["Status"])
	}
	return statuses, nil
}

// AllocStatusesRescheduled is a helper function that pulls
// out client statuses only from rescheduled allocs.
func AllocStatusesRescheduled(jobID string) ([]string, error) {

	out, err := Command("nomad", "job", "status", "-verbose", jobID)
	if err != nil {
		return nil, fmt.Errorf("nomad job status failed: %w", err)
	}

	section, err := GetSection(out, "Allocations")
	if err != nil {
		return nil, fmt.Errorf("could not find Allocations section: %w", err)
	}

	allocs, err := ParseColumns(section)
	if err != nil {
		return nil, fmt.Errorf("could not parse Allocations section: %w", err)
	}

	statuses := []string{}
	for _, alloc := range allocs {

		allocID := alloc["ID"]

		// reschedule tracker isn't exposed in the normal CLI output
		out, err := Command("nomad", "alloc", "status", "-json", allocID)
		if err != nil {
			return nil, fmt.Errorf("nomad alloc status failed: %w", err)
		}

		dec := json.NewDecoder(strings.NewReader(out))
		alloc := &api.Allocation{}
		err = dec.Decode(alloc)
		if err != nil {
			return nil, fmt.Errorf("could not decode alloc status JSON: %w", err)
		}

		if (alloc.RescheduleTracker != nil &&
			len(alloc.RescheduleTracker.Events) > 0) || alloc.FollowupEvalID != "" {
			statuses = append(statuses, alloc.ClientStatus)
		}
	}
	return statuses, nil
}
