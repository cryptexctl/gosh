package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type JobState int

const (
	JobRunning JobState = iota
	JobStopped
	JobDone
	JobKilled
)

func (s JobState) String() string {
	switch s {
	case JobRunning:
		return "Running"
	case JobStopped:
		return "Stopped"
	case JobDone:
		return "Done"
	case JobKilled:
		return "Killed"
	default:
		return "Unknown"
	}
}

type Job struct {
	ID       int
	PID      int
	Command  string
	State    JobState
	Started  time.Time
	Finished *time.Time
	ExitCode int
	Process  *os.Process
	Cmd      *exec.Cmd
}

type Manager struct {
	jobs   map[int]*Job
	nextID int
	mu     sync.RWMutex
}

func New() *Manager {
	return &Manager{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

func (m *Manager) Add(cmd *exec.Cmd, command string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &Job{
		ID:      m.nextID,
		PID:     cmd.Process.Pid,
		Command: command,
		State:   JobRunning,
		Started: time.Now(),
		Process: cmd.Process,
		Cmd:     cmd,
	}

	m.jobs[m.nextID] = job
	m.nextID++

	go m.monitor(job)

	return job
}

func (m *Manager) Get(id int) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.jobs[id]
}

func (m *Manager) GetByPID(pid int) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, job := range m.jobs {
		if job.PID == pid {
			return job
		}
	}
	return nil
}

func (m *Manager) List() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (m *Manager) Running() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		if job.State == JobRunning {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (m *Manager) Stopped() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		if job.State == JobStopped {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (m *Manager) Kill(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State != JobRunning && job.State != JobStopped {
		return fmt.Errorf("job %d is not running", id)
	}

	if job.Process != nil {
		err := job.Process.Signal(syscall.SIGTERM)
		if err != nil {
			err = job.Process.Kill()
		}

		if err == nil {
			job.State = JobKilled
			now := time.Now()
			job.Finished = &now
		}

		return err
	}

	return fmt.Errorf("no process for job %d", id)
}

func (m *Manager) Stop(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State != JobRunning {
		return fmt.Errorf("job %d is not running", id)
	}

	if job.Process != nil {
		err := job.Process.Signal(syscall.SIGSTOP)
		if err == nil {
			job.State = JobStopped
		}
		return err
	}

	return fmt.Errorf("no process for job %d", id)
}

func (m *Manager) Continue(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State != JobStopped {
		return fmt.Errorf("job %d is not stopped", id)
	}

	if job.Process != nil {
		err := job.Process.Signal(syscall.SIGCONT)
		if err == nil {
			job.State = JobRunning
		}
		return err
	}

	return fmt.Errorf("no process for job %d", id)
}

func (m *Manager) Foreground(id int) error {
	job := m.Get(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State == JobStopped {
		if err := m.Continue(id); err != nil {
			return err
		}
	}

	if job.Cmd != nil {
		return job.Cmd.Wait()
	}

	return nil
}

func (m *Manager) Background(id int) error {
	job := m.Get(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State == JobStopped {
		return m.Continue(id)
	}

	return nil
}

func (m *Manager) Clean() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, job := range m.jobs {
		if job.State == JobDone || job.State == JobKilled {
			delete(m.jobs, id)
		}
	}
}

func (m *Manager) Wait() {
	for {
		running := m.Running()
		if len(running) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Manager) WaitJob(id int) error {
	job := m.Get(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.Cmd != nil {
		return job.Cmd.Wait()
	}

	return nil
}

func (m *Manager) monitor(job *Job) {
	if job.Cmd == nil {
		return
	}

	err := job.Cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	job.Finished = &now

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				job.ExitCode = status.ExitStatus()
			}
		}
		job.State = JobKilled
	} else {
		job.ExitCode = 0
		job.State = JobDone
	}
}

func (m *Manager) Print() {
	jobs := m.List()
	if len(jobs) == 0 {
		fmt.Println("No jobs")
		return
	}

	fmt.Printf("%-3s %-8s %-10s %-8s %s\n", "ID", "PID", "STATE", "TIME", "COMMAND")
	fmt.Printf("%-3s %-8s %-10s %-8s %s\n", "---", "-----", "-----", "----", "-------")

	for _, job := range jobs {
		duration := time.Since(job.Started)
		if job.Finished != nil {
			duration = job.Finished.Sub(job.Started)
		}

		fmt.Printf("%-3d %-8d %-10s %-8s %s\n",
			job.ID,
			job.PID,
			job.State.String(),
			formatDuration(duration),
			job.Command,
		)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.jobs)
}

func (m *Manager) RunningCount() int {
	return len(m.Running())
}

func (m *Manager) StoppedCount() int {
	return len(m.Stopped())
}
