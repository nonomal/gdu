package main

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"sync"
)

// CurrentProgress struct
type CurrentProgress struct {
	mutex           *sync.Mutex
	currentItemName string
	itemCount       int
	totalSize       int64
	done            bool
}

// ProcessDir analyzes given path
func ProcessDir(path string, progress *CurrentProgress) *File {
	concurrencyLimitChannel := make(chan bool, 2*runtime.NumCPU())
	var wait sync.WaitGroup
	dir := processDir(path, progress, concurrencyLimitChannel, &wait)
	wait.Wait()
	dir.UpdateStats()
	return dir
}

func processDir(path string, progress *CurrentProgress, concurrencyLimitChannel chan bool, wait *sync.WaitGroup) *File {
	var file *File
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}

	dir := File{
		name:      filepath.Base(path),
		path:      path,
		isDir:     true,
		itemCount: 1,
		files:     make([]*File, 0, len(files)),
	}

	var mutex sync.Mutex
	var totalSize int64

	for _, f := range files {
		entryPath := filepath.Join(path, f.Name())

		if f.IsDir() {
			wait.Add(1)
			go func() {
				concurrencyLimitChannel <- true
				file = processDir(entryPath, progress, concurrencyLimitChannel, wait)
				file.parent = &dir
				mutex.Lock()
				dir.files = append(dir.files, file)
				mutex.Unlock()
				<-concurrencyLimitChannel
				wait.Done()
			}()
		} else {
			file = &File{
				name:      f.Name(),
				path:      entryPath,
				size:      f.Size(),
				itemCount: 1,
				parent:    &dir,
			}
			totalSize += f.Size()

			mutex.Lock()
			dir.files = append(dir.files, file)
			mutex.Unlock()
		}
	}

	progress.mutex.Lock()
	progress.currentItemName = path
	progress.itemCount += len(files)
	progress.totalSize += totalSize
	progress.mutex.Unlock()

	return &dir
}
