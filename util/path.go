package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func MakeFilePath(path string) error {
	dir := filepath.Dir(path)
	if dir == "" {
		return fmt.Errorf("the specified file(%s) path is not valid", path)
	}

	isExist, err := Exists(dir)
	if err != nil {
		return err
	}

	if isExist {
		return nil
	}

	return os.Mkdir(dir, 0644)
}
