// Package util is a grab bag for stuff that needs to go elsewhere.
package util

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func Open(path string, mode os.FileMode) (file io.Writer) {

	var err error
	file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		fmt.Printf("warning: %s\n", err.Error())
		file = io.Discard
	}

	return
}

func Close(file io.Writer) {

	actually, ok := file.(*os.File)
	if ok {
		actually.Close()
	}
}

func Load(obj any, path string) (err error) {

	data, err := os.ReadFile(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to read from %s", path)
		return
	}

	err = yaml.Unmarshal(data, obj)
	err = errors.Wrapf(err, "failed to unmarshal")
	return
}

func Save(obj any, path string, mode os.FileMode) (err error) {

	data, err := yaml.Marshal(obj)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal")
		return
	}

	err = os.WriteFile(path, data, mode)
	err = errors.Wrapf(err, "failed to write to %s", path)
	return
}
