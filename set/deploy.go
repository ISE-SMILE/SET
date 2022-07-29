package set

import (
	"fmt"
	"github.com/google/martian/log"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//Assumptions:
//1.) we have a folders workloads/{go,python}/ containing the base function files
//2.) we have a folders functions/{aws,ow,gcf,azf}/{go,python}/ containg a Makefile with three rules: deploy, remove, update,info

const goFiles = "function.go"
const pyFiles = "bencher.py Pipfile"

const targets = "aws ow gcf azf"

type MakefileDeployment struct {
}

func (m MakefileDeployment) Deploy(d Deployment) (string, error) {
	err := copyBase()
	if err != nil {
		return "", err
	}
	msg, err := run(d, "deploy")
	if err != nil {
		return "", err
	}

	log.Infof("deployed %s", msg)

	out, err := run(d, "info")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m MakefileDeployment) Remove(d Deployment) error {
	_, err := run(d, "remove")
	return err
}

func (m MakefileDeployment) Change(d Deployment) error {
	_, err := run(d, "update")
	return err
}

func run(d Deployment, rule string) ([]byte, error) {
	cmd, err := makeCmd(d, rule)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("failed to deploy cause:%s", string(output))
		return nil, err
	}

	return output, nil
}

func makeCmd(d Deployment, rule string) (*exec.Cmd, error) {
	if _, err := os.Stat(filepath.Join(d.Source, "Makefile")); err != nil {
		return nil, fmt.Errorf("the Makefile is missing at %s", d.Source)
	}
	cmd := exec.Command("make", rule)
	cmd.Dir = d.Source
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("MEM=%d", d.FunctionMemory))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TIMEOUT=%d", int(math.Ceil(d.FunctionTimeout.Seconds()))))
	cmd.Env = append(cmd.Env, fmt.Sprintf("REGION=%s", d.FunctionRegion))
	return cmd, nil
}

func copyBase() error {
	err := copyFiles(
		strings.Split(goFiles, " "),
		strings.Split(targets, " "),
		"go",
		"bencher",
	)
	if err != nil {
		return err
	}

	err = copyFiles(
		strings.Split(pyFiles, " "),
		strings.Split(targets, " "),
		"python",
		"",
	)
	if err != nil {
		return err
	}

	return nil
}

func copyFiles(files, targets []string, runtime, prefix string) error {
	for _, f := range files {
		for _, t := range targets {
			src := filepath.Join("workloads", runtime, f)
			dest := filepath.Join("functions", t, runtime, prefix, f)
			err := copy(src, dest)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}
	return nil
}
