package model

import (
	"errors"
	"gitlab.com/cretz/fusty/config"
)

type CommandSet struct {
	Commands []string `json:"commands"`
}

func NewCommandSetFromConfig(conf *config.JobCommand) (*CommandSet, error) {
	if len(conf.Inline) == 0 {
		return nil, errors.New("No commands in set")
	}
	return &CommandSet{Commands: conf.Inline}, nil
}

func (c *CommandSet) DeepCopy() *CommandSet {
	ret := &CommandSet{Commands: make([]string, len(c.Commands))}
	copy(ret.Commands, c.Commands)
	return ret
}
