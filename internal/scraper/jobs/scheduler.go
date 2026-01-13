package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

// GetPeriodicJobs returns the periodic jobs to schedule
func GetPeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		// Run full scrape daily at 2 AM UTC
		river.NewPeriodicJob(
			river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return ScrapeAllJobArgs{}, nil
			},
			&river.PeriodicJobOpts{
				RunOnStart: false,
			},
		),
	}
}
