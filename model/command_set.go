package model

import (
	"errors"
	"gitlab.com/cretz/fusty/config"
)

type CommandSet struct {
	Commands []string
}

func NewCommandSetFromConfig(conf *config.JobCommand) (*CommandSet, error) {
	if len(conf.Inline) == 0 {
		return nil, errors.New("No commands in set")
	}
	return &CommandSet{Commands: conf.Inline}, nil
}
