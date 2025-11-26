// Package util is a grab bag for stuff that needs to go elsewhere.
package util

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Todo: copied from dft, modularize!

func OpenLog(path string, mode os.FileMode) (file io.Writer) {

	var err error
	file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		fmt.Printf("warning: %s\n", err.Error())
		file = io.Discard
	}

	return
}

func CloseLog(file io.Writer) {

	actually, ok := file.(*os.File)
	if ok {
		actually.Close()
	}
}

func LoadConfig(cfg any, path string) (err error) {

	// Todo: could check that loaded config is different from sample
	//       but need to cleanup first yeah

	data, err := os.ReadFile(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to read from %s", path)
		return
	}

	err = yaml.Unmarshal(data, cfg)
	err = errors.Wrapf(err, "failed to unmarshal")
	return
}

func WriteConfig(cfg any, path string, mode os.FileMode) (err error) {

	data, err := yaml.Marshal(cfg)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal")
		return
	}

	err = os.WriteFile(path, data, mode)
	err = errors.Wrapf(err, "failed to write to %s", path)
	return
}

func SampleConfig(data []byte, path string, mode os.FileMode) (err error) {

	// Todo: think about just writing the sample after failing to read in load

	_, err = os.Stat(path)
	if err == nil {
		return // already have a cfg
	}

	err = os.WriteFile(path, data, mode)
	err = errors.Wrapf(err, "failed to write to %s", path)
	return
}
