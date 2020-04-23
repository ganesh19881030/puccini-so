package components

import (
	"fmt"
	"strconv"
	"sync"
)

var Processed int

type Workflow struct {
	Name      string              `json:"name" yaml:"name"`
	Processes map[string]*Process `json:"processes" yaml:"processes"`
	StepCount int                 `json:"stepcount" yaml:"stepcount"`
}

func NewWorkflow(name string, count int) *Workflow {
	wf := &Workflow{
		Name:      name,
		Processes: map[string]*Process{},
		StepCount: count,
	}

	return wf
}

func (wf *Workflow) AddProc(proc *Process) {
	//wf.processes = append(wf.processes, proc)
	wf.Processes[proc.Name] = proc
}

func (wf *Workflow) Run(wg *sync.WaitGroup) *WfError {
	var err *WfError

	Processed = 0
	procs := wf.GetStartProcesses()
	processes := make([]string, 0)
	for _, proc := range procs {
		processes = append(processes, proc.GetName())
	}
	wg.Add(1)
	//fmt.Println("started processes ", processes)
	go execute(wf, processes, wg)
	//fmt.Println("ended processes ", processes)
	return err

}

func (wf *Workflow) GetProcs() map[string]*Process {
	return wf.Processes
}

func (wf *Workflow) GetStartProcesses() []*Process {
	procs := make([]*Process, 0)
	for _, proc := range wf.Processes {
		if proc.IsStartProcess() {
			procs = append(procs, proc)
		}
	}
	return procs
}

func (wf *Workflow) GetProcByName(name string) *Process {
	return wf.Processes[name]
}

func (wf *Workflow) SetCount(num int) {
	wf.StepCount = num
}

func (wf *Workflow) canExecute(name string) bool {
	proc := wf.GetProcByName(name)
	if proc.IsComplete() {
		return false
	}

	procs := getCallingProcs(wf.GetProcs(), name)
	for _, p := range procs {
		if !p.IsComplete() {
			return false
		}
	}
	return true
}

func processStep(wf *Workflow, curr *Process, wg *sync.WaitGroup) {
	if curr != nil {
		//fmt.Println(curr.GetName())
		err := curr.Run()
		if err == nil && len(curr.GetNextSuccesses()) > 0 {
			wg.Add(1)
			//fmt.Println("started processes *******", curr.GetNextSuccesses())
			go execute(wf, curr.GetNextSuccesses(), wg)
			//fmt.Println("ended processes ********", curr.GetNextSuccesses())
		} else if err != nil && len(curr.GetNextFailures()) > 0 {
			wg.Add(1)
			go execute(wf, curr.GetNextFailures(), wg)
		}
	}
	return
}

func execute(wf *Workflow, processes []string, wg *sync.WaitGroup) {
	defer wg.Done()
	procs := make([]string, 0)
	c1 := make(chan []string)
	for _, next := range processes {
		proc := wf.GetProcByName(next)
		if proc.IsComplete() {
			continue
		} else if wf.canExecute(proc.GetName()) {
			wg.Add(1)
			fmt.Println("start ====>>", proc.GetName())
			go runProcess(proc, c1, wg)
			p := <-c1
			fmt.Println("end =========================================================>>")
			//fmt.Println(p)
			procs = append(procs, p...)
		} else {
			procs = append(procs, proc.GetName())
		}

	}

	if len(procs) > 0 {
		p := removeDuplicates(procs)
		wg.Add(1)
		//fmt.Println("started processes *************", p)
		go execute(wf, p, wg)
		//fmt.Println("ended processes ************", p)
	} else {
		fmt.Println("Number of steps processed: " + strconv.Itoa(Processed))
	}

	return
}

func removeDuplicates(procs []string) []string {
	ps := make([]string, 0)
	for _, proc := range procs {
		if !findProc(proc, ps) {
			ps = append(ps, proc)
		}
	}
	return ps

}

func runProcess(proc *Process, c chan []string, wg *sync.WaitGroup) {
	defer wg.Done()
	procs := make([]string, 0)
	//fmt.Print("Running process =======>>>", proc.GetName())
	Processed++
	err := proc.Run()

	if err == nil {
		proc.SetComplete(true)
		if len(proc.GetNextSuccesses()) > 0 {
			for _, success := range proc.GetNextSuccesses() {
				procs = append(procs, success)
			}
		}
	}
	c <- procs
}

func findProc(proc string, procs []string) bool {
	for _, elem := range procs {
		if elem == proc {
			return true
		}
	}
	return false
}

func getCallingProcs(procs map[string]*Process, name string) []*Process {
	var isCalled bool
	prevProcs := make([]*Process, 0)
	for _, proc := range procs {
		isCalled = hasProc(proc.GetNextSuccesses(), name)
		if isCalled {
			prevProcs = append(prevProcs, proc)
		}
	}

	return prevProcs
}

func hasProc(procs []string, name string) bool {
	for _, proc := range procs {
		if proc == name {
			return true
		}
	}

	return false
}
