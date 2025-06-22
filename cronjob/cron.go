package cronjob

import (
	"github.com/maxliu9403/common/logger"
	"github.com/robfig/cron/v3"
)

type Job struct {
	cronJobs *cron.Cron
}

var CronJobs = NewCronJob()

func NewCronJob() *Job {
	return &Job{cronJobs: cron.New()}
}

func (j *Job) Start() {
	j.cronJobs.Start()
}

func (j *Job) Stop() {
	j.cronJobs.Stop()
}

func (j *Job) Terminate() {
	j.cronJobs.Stop()
	j.Clear()
}

func (j *Job) ListEntries() []cron.Entry {
	return j.cronJobs.Entries()
}

func (j *Job) RemoveJobByID(id cron.EntryID) {
	j.cronJobs.Remove(id)
}

func (j *Job) HasRunningJobs() bool {
	for _, entry := range j.cronJobs.Entries() {
		if !entry.Next.IsZero() {
			return true
		}
	}
	return false
}

func (j *Job) Clear() {
	for _, entry := range j.cronJobs.Entries() {
		j.cronJobs.Remove(entry.ID)
	}
	logger.Infof("clearing the cron job successfully, the current number of tasks：%d", len(j.cronJobs.Entries()))
}

func (j *Job) AddJob(spec string, job cron.Job) (cron.EntryID, error) {
	entryID, err := j.cronJobs.AddJob(spec, job)
	if err != nil {
		return entryID, err
	}
	logger.Infof("adding cron job (EntryId=%d) succeeded，the current number of tasks：%d", entryID, len(j.cronJobs.Entries()))
	return entryID, err
}
