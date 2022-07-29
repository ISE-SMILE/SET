package set

import "time"

type Platform interface {
	Deploy(Deployment) (string, error)
	Change(Deployment) error
	Remove(Deployment) error
}

type Deployment struct {
	Source string `json:"source,omitempty" yaml:"source"`

	FunctionRuntime string        `json:"runtime,omitempty" yaml:"runtime"`
	FunctionMemory  Unit          `json:"memory,omitempty" yaml:"memory"`
	FunctionTimeout time.Duration `json:"timeout,omitempty" yaml:"timeout"`
	FunctionRegion  string        `json:"region,omitempty" yaml:"region"`
}
