package utils

import (
	"os"
	"path/filepath"
)

func InferRootDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var infer func(dir string) string
	infer = func(dir string) string {
		if exists(dir + "/conf") {
			return dir
		}

		return infer(filepath.Dir(dir))
	}

	return infer(cwd)
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
