package job

import (
	"fmt"

	"github.com/riverqueue/river"
	"github.com/robfig/cron/v3"
)

// the delivery window is the whole current minute, so a late tick still lands
// and RunOnStart covers a worker that was down when the minute began
func EveryMinuteSchedule() (cron.Schedule, error) {
	schedule, err := cron.ParseStandard("* * * * *")
	if err != nil {
		return nil, fmt.Errorf("parse cron schedule: %w", err)
	}
	return schedule, nil
}

func NewWorkers(dispatch *DispatchDigestsWorker, send *SendDigestWorker) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, dispatch)
	river.AddWorker(workers, send)
	return workers
}

func DispatchDigestsPeriodicJob(schedule cron.Schedule) *river.PeriodicJob {
	return river.NewPeriodicJob(
		schedule,
		func() (river.JobArgs, *river.InsertOpts) {
			return DispatchDigestsArgs{}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: true},
	)
}
