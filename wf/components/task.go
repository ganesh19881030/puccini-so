package components

type Task struct {
	Description string `json:"description" yaml:"description"`
	Sequence    int    `json:"sequence" yaml:"sequence"`
	//Properties  interface{}     `json:"properties" yaml:"properties"`
	customExec func() *WfError
}

//func NewTask(proc *Process, desc string, seq int, props interface{}, fn func() *WfError) *Task {
func NewTask(proc *Process, desc string, seq int, fn func() *WfError) *Task {
	task := &Task{
		//name:       name,
		Sequence:    seq,
		Description: desc,
		//Properties:  props,
		customExec: fn,
	}
	proc.AddTask(task)

	return task
}

func (task *Task) Execute() *WfError {
	err := task.customExec()
	return err
}
