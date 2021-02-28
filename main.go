package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Store is the collection of files to monitor. Along with the filename the last time the file was
// changed is stored inside the store. Store is just a wrapper around the build-in map-type.
type Store map[string]time.Time

// Check compares the stored timestamp with the current timestamp for each file in the store. If a
// file has changed the filename gets send through the given channel. The channel gets closed when
// all files were processed.
func (s Store) Check(c chan<- string) {
	for k, v := range s {
		currentTimeStamp, err := GetTimeStamp(k)
		if err != nil {
			// TODO: log this error the stdout
			continue
		}

		// If the current timeStamp is bigger (more recent date) than the file has been changed.
		if currentTimeStamp.Sub(v) > 0 {
			s[k] = currentTimeStamp
			c <- k
		}
	}

	// Done, close the channel
	close(c)
}

// NewStore creates a new Store containing all files to monitor. The last time a file was edited is
// stored alongside the name of the file.
func NewStore(path string) Store {
	store := make(Store)

	dirs, err := FilePathWalkDir(path)
	if err != nil {
		panic(err)
	}

	// fill store with files
	for _, dir := range dirs {
		timeStamp, err := GetTimeStamp(dir)
		if err != nil {
			fmt.Printf("Could not monitor file: %s", dir)
			continue
		}

		store[dir] = timeStamp
	}

	return store
}

var store Store

func init() {
	store = NewStore(os.Args[1])
}

func main() {
	for {
		c := make(chan string)
		go store.Check(c)

		for f := range c {
			fmt.Printf("File changed: %s\n", f)
			out, err := RunCmd(os.Args[2], os.Args[3:]...)
			if err != nil {
				fmt.Printf("Error executing command (%s): %s", os.Args[2:], err)
			} else {
				// FIXME: when the command is like echo "foo" >> test, out is also written to the file and
				// not the stdout
				os.Stdout.Write(out)
			}
		}
	}
}

func GetTimeStamp(filePath string) (time.Time, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}

	return fileInfo.ModTime(), nil
}

// FilePathWalkDir returns all files inside a directory and all files each subdirectory. This
// happens recursive, so that all files in any directory under the given root-directory get listed.
func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// RunCmd executes a given command with the given arguments. The stdout and stderr are returned if
// there was no error.
func RunCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return stdoutStderr, nil
}
