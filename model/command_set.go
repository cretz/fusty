package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"regexp"
	"strings"
)

type CommandSet struct {
	Generic  *CommandSetCommand   `json:"-"`
	Commands []*CommandSetCommand `json:"commands"`
}

func NewDefaultCommandSet() *CommandSet {
	return &CommandSet{Generic: NewDefaultCommandSetCommand(), Commands: []*CommandSetCommand{}}
}

func (c *CommandSet) ApplyConfig(conf *config.Job) {
	if conf.CommandGeneric != nil {
		c.Generic.ApplyConfig(conf.CommandGeneric)
	}
	for _, cmd := range conf.Commands {
		setCmd := c.Generic.DeepCopy()
		setCmd.ApplyConfig(cmd)
		c.Commands = append(c.Commands, setCmd)
	}
}

func (c *CommandSet) Validate() []error {
	errs := []error{}
	if len(c.Commands) == 0 {
		errs = append(errs, errors.New("No commands in set"))
	}
	for _, cmd := range c.Commands {
		for _, err := range cmd.Validate() {
			errs = append(errs, fmt.Errorf("Command '%v' invalid: %v", cmd.Command, err))
		}
	}
	return errs
}

func (c *CommandSet) DeepCopy() *CommandSet {
	ret := &CommandSet{Commands: []*CommandSetCommand{}}
	for _, cmd := range c.Commands {
		ret.Commands = append(ret.Commands, cmd.DeepCopy())
	}
	return ret
}

type CommandSetCommand struct {
	Command       string   `json:"command"`
	Expect        []string `json:"expect"`
	ExpectNot     []string `json:"expect_not"`
	Timeout       int      `json:"timeout"`
	ImplicitEnter bool     `json:"implicit_enter"`
}

func NewDefaultCommandSetCommand() *CommandSetCommand {
	return &CommandSetCommand{
		Expect:        []string{},
		ExpectNot:     []string{},
		Timeout:       120,
		ImplicitEnter: true,
	}
}

func (c *CommandSetCommand) ApplyConfig(conf *config.JobCommand) {
	if conf.Command != "" {
		c.Command = conf.Command
	}
	// We don't pre-compile the regex because it's sent over the wire
	for _, re := range conf.Expect {
		c.Expect = append(c.Expect, c.sanitizeRegex(re))
	}
	for _, re := range conf.ExpectNot {
		c.ExpectNot = append(c.ExpectNot, c.sanitizeRegex(re))
	}
	if conf.Timeout != nil {
		c.Timeout = *conf.Timeout
	}
	if conf.ImplicitEnter != nil {
		c.ImplicitEnter = *conf.ImplicitEnter
	}
}

func (c *CommandSetCommand) sanitizeRegex(regex string) string {
	// We implicitly prepend .* if ^ isn't there and same at end for lack of $
	// TODO: maybe this should check that the first char isn't a regex char instead?
	if !strings.HasPrefix(regex, "^") {
		regex = ".*" + regex
	}
	if !strings.HasSuffix(regex, "$") {
		regex += ".*"
	}
	return regex
}

// Validate after all configs applied
func (c *CommandSetCommand) Validate() []error {
	errs := []error{}
	if c.Command == "" {
		errs = append(errs, errors.New("Command is empty"))
	}
	for _, exp := range c.Expect {
		if _, err := regexp.Compile(exp); err != nil {
			errs = append(errs, fmt.Errorf("Invalid regex '%v': %v", exp, err))
		}
	}
	for _, exp := range c.ExpectNot {
		if _, err := regexp.Compile(exp); err != nil {
			errs = append(errs, fmt.Errorf("Invalid regex '%v': %v", exp, err))
		}
	}
	// By rule, the timeout can only be 0 if there aren't any expectations
	if c.Timeout == 0 && (len(c.Expect) != 0 || len(c.ExpectNot) != 0) {
		errs = append(errs, errors.New("Timeout can only be 0 when there are no expectations"))
	}
	return errs
}

func (c *CommandSetCommand) DeepCopy() *CommandSetCommand {
	ret := &CommandSetCommand{
		Command:       c.Command,
		Expect:        make([]string, len(c.Expect)),
		ExpectNot:     make([]string, len(c.ExpectNot)),
		Timeout:       c.Timeout,
		ImplicitEnter: c.ImplicitEnter,
	}
	copy(ret.Expect, c.Expect)
	copy(ret.ExpectNot, c.ExpectNot)
	return ret
}
