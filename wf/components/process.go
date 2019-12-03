package components

type Process struct {
	Name      string   `json:"name" yaml:"name"`
	Tasks     []*Task  `json:"tasks" yaml:"tasks"`
	IsStart   bool     `json:"start" yaml:"start"`
	Complete  bool     `json:"complete" yaml:"complete"`
	Successes []string `json:"success" yaml:"success"`
	Failures  []string `json:"failure" yaml:"failure"`
}

/*type Process struct {
	Name    string
	Tasks   []*Task
	IsStart   bool
	Successes []string
	Failures []string
}*/

func NewProcess(wf *Workflow, name string, nextSuccesses []string, nextFailures []string) *Process {
	proc := &Process{
		Name:      name,
		Tasks:     []*Task{},
		IsStart:   false,
		Complete:  false,
		Successes: nextSuccesses,
		Failures:  nextFailures,
	}
	wf.AddProc(proc)

	return proc
}

func (proc *Process) AddTask(task *Task) {
	proc.Tasks = append(proc.Tasks, task)
}

func (proc *Process) SetStart(flag bool) {
	proc.IsStart = flag
}

func (proc *Process) GetName() string {
	return proc.Name
}

func (proc *Process) IsStartProcess() bool {
	return proc.IsStart
}

func (proc *Process) IsComplete() bool {
	return proc.Complete
}

func (proc *Process) Run() *WfError {
	tasks := proc.Tasks
	for _, task := range tasks {
		err := task.Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func (proc *Process) GetNextSuccesses() []string {
	return proc.Successes
}

func (proc *Process) GetNextFailures() []string {
	return proc.Failures
}

func (proc *Process) SetComplete(flag bool) {
	proc.Complete = flag
}
