package stdin

import (
	"errors"
	"io/ioutil"
	"os"
	"sync"
)

var (
	stdinErr   error
	stdinBytes []byte
	stdinLock  sync.RWMutex
)

// GetBytes returns the contents of os.Stdin which can only be read once
func GetBytes() ([]byte, error) {
	stdinLock.RLock()

	if stdinBytes == nil {
		stdinLock.RUnlock()

		stdinLock.Lock()
		defer stdinLock.Unlock()

		// Multiple readers could be at stdinLock.Lock()
		// Check condition again
		if stdinBytes == nil {
			stdinBytes, stdinErr = readStdin()
		}
	} else {
		stdinLock.RUnlock()
	}

	return stdinBytes, stdinErr
}

func readStdin() ([]byte, error) {

	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		err = errors.New("os.Stdin is not a valid character device")
		return nil, err
	}

	return ioutil.ReadAll(os.Stdin)
}
