package main

import (
	"time"
	"workerChannel/worker"
)

func main() {
	worker.InitWorkerPool()
	go func() {
		worker.Dispatch()
	}()
	go func() {
		var count = int64(1);
		for {
			t := time.NewTimer(time.Second * 1)
			select {
				case <-t.C:
					worker.JobQueue<-count
					count++
			}
		}
	}()


	for{
		;
	}

}