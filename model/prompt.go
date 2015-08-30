package model

import (
	"errors"
	"gitlab.com/cretz/fusty/config"
)

type Prompt struct {
	EndsWith string `json:"ends_with"`
}

func NewDefaultPrompt() *Prompt {
	return &Prompt{}
}

func (p *Prompt) ApplyConfig(conf *config.Prompt) error {
	if conf.EndsWith != "" {
		p.EndsWith = conf.EndsWith
	}
	return nil
}

func (d *Prompt) Validate() []error {
	return []error{errors.New("Not implemented")}
}
