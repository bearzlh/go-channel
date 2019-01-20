package worker

import (
	"fmt"
)

type Worker struct {
	ID      int
	RepJobs chan int64
	quit    chan bool
}

type workerPool struct {
	workerChan chan *Worker
	workerList []*Worker
}

func (w *Worker) handleJob(jobId int64) {
	fmt.Printf("%d", jobId)
}

func (w *Worker) Start() {
	go func() {
		for {
			WorkerPool.workerChan <- w
			select {
			case jobID := <-w.RepJobs:
				fmt.Printf("worker: %d, will handle job: %d", w.ID, jobID)
				w.handleJob(jobID)
			case q := <-w.quit:
				if q {
					fmt.Printf("worker: %d, will stop.", w.ID)
					return
				}
			}
		}
	}()
}

func NewWorker(i int) *Worker {
	return &Worker{i, nil, nil}
}

var WorkerPool *workerPool
var JobQueue = make(chan int64);

func InitWorkerPool() error {
	n := 3
	WorkerPool = &workerPool{
		workerChan: make(chan *Worker, n),
		workerList: make([]*Worker, 0, n),
	}
	for i := 0; i < n; i++ {
		worker := NewWorker(i)
		WorkerPool.workerList = append(WorkerPool.workerList, worker)
		worker.Start()
		fmt.Printf("worker %d started\n", worker.ID)
	}
	return nil
}

func Dispatch() {
	for {
		select {
		case job := <-JobQueue:
			go func(jobID int64) {
				fmt.Printf("Trying to dispatch job: %d\n", jobID)
				worker := <-WorkerPool.workerChan
				worker.RepJobs <- jobID
			}(job)
		}
	}
}

func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}