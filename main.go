package main

import "fmt"

type Task struct {
	Name string
	Run  func()
}

var taskQueue chan Task

func worker(id int) {
	for task := range taskQueue {
		fmt.Printf("[Worker %d] Task started: %s\n", id, task.Name)
		task.Run()
		fmt.Printf("[Worker %d] Task Complited: %s\n", id, task.Name)
	}
}

var config Data

func main() {
	var err error

	maxTasks := 6

	taskQueue = make(chan Task, maxTasks)

	for i := 0; i < maxTasks; i++ {
		go worker(i)
	}

	config, err = NewConfigReader()
	if err != nil {
		fmt.Println("Error reading configuration file:", err)
		return
	}

	server := NewServer("‎", config.Port.Value, 5, 10)
	server.StartServer()
}
