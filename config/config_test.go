package config_test

import (
	"gitlab.com/cretz/fusty/config"
	"reflect"
	"testing"
)

func TestJSONUnmarshal(t *testing.T) {
	json := `{
	  "ip": "127.0.0.1",
	  "port": 9400,
	  "job_store": {
	  	"type": "local",
	  	"local": {
	  	  "job_generics": {
	  	  	"foo": {
	  	  	  "schedule": {"cron": "0 0 * * * * *"},
	  	  	  "commands": [
	  	  	    {"command": "command1"},
	  	  	    {"command": "command2"}
	  	  	  ]
	  	  	},
	  	  	"linux_vm_job": {
	  	  	  "command_generic": {
	  	  	    "expect": ["vagrant@linux-vm:"],
	  	  	    "expect_not": ["No such file or directory"]
	  	  	  }
	  	  	}
	  	  },
	  	  "jobs": {
	  	  	"bar": {"generic": "foo"},
	  	  	"show_config": {
	  	  	  "generic": "linux_vm_job",
	  	  	  "schedule": {"cron": "*/3 * * * * * *"},
	  	  	  "commands": [{"command": "cat /vagrant/sample-txt"}],
	  	  	  "scrubbers": [{
	  	  	    "type": "simple",
	  	  	    "search": "multilink bundle-name authenticated",
	  	  	    "replace": "multilink bundle-name {{replace_authenticated}}"
	  	  	  }],
	  	  	  "template_values": {"replace_authenticated": "job-level"}
	  	  	}
	  	  }
	  	}
	  },
	  "device_store": {
	  	"type": "local",
	  	"local": {
	  	  "device_generics": {
	  	    "baz": {},
	  	    "linux_vm_base": {
	  	      "host": "127.0.0.1",
	  	      "protocol": {
	  	        "type": "ssh",
	  	        "ssh": {"port": 3222}
	  	      },
	  	      "credentials": {
	  	      	"user": "vagrant",
	  	      	"pass": "vagrant"
	  	      }
	  	    }
	  	  },
	  	  "devices": {
	  	  	"qux": {
	  	  	  "host": "127.0.0.1",
	  	  	  "generic": "baz",
	  	  	  "protocol": {"type": "ssh"}
	  	  	},
	  	  	"local_linux_vm": {
	  	  	  "generic": "linux_vm_base",
	  	  	  "jobs": {
	  	  	  	"show_config": {
	  	  	  	  "template_values": {"replace_authenticated": "device-level"}
	  	  	  	}
	  	  	  }
	  	  	}
	  	  }
	  	}
	  },
	  "data_store": {
	    "type": "git",
	    "git": {
	      "url": "someurl1",
	      "pool_size": 1,
	      "data_dir": "somedir1",
	      "user": {
	      	"friendly_name": "John Doe",
	      	"email": "jdoe@example.com"
	      }
	    }
	  }
	}`
	assertValidConfig(t, json, config.JSONFormat)
}

func TestYAMLUnmarshal(t *testing.T) {
	yaml := `
ip: 127.0.0.1
port: 9400
data_store:
  type: git
  git:
    url: someurl1
    user:
      friendly_name: John Doe
      email: jdoe@example.com
    pool_size: 1
    data_dir: somedir1
job_store:
  type: local
  local:
    job_generics:
      foo:
        schedule:
          cron: 0 0 * * * * *
        commands:
        - command: command1
        - command: command2
      linux_vm_job:
        command_generic:
          expect:
          - 'vagrant@linux-vm:'
          expect_not:
          - No such file or directory
    jobs:
      bar:
        generic: foo
      show_config:
        generic: linux_vm_job
        schedule:
          cron: '*/3 * * * * * *'
        commands:
        - command: cat /vagrant/sample-txt
        scrubbers:
        - type: simple
          search: multilink bundle-name authenticated
          replace: multilink bundle-name {{replace_authenticated}}
        template_values:
          replace_authenticated: job-level
device_store:
  type: local
  local:
    device_generics:
      baz: {}
      linux_vm_base:
        host: 127.0.0.1
        protocol:
          type: ssh
          ssh:
            port: 3222
        credentials:
          user: vagrant
          pass: vagrant
    devices:
      local_linux_vm:
        generic: linux_vm_base
        jobs:
          show_config:
            template_values:
              replace_authenticated: device-level
      qux:
        generic: baz
        host: 127.0.0.1
        protocol:
          type: ssh`
	assertValidConfig(t, yaml, config.YAMLFormat)
}

func TestTOMLUnmarshal(t *testing.T) {
	toml := `
ip = "127.0.0.1"
port = 9400

[job_store]
type = "local"
  [job_store.local]
    [job_store.local.job_generics.foo]
      [job_store.local.job_generics.foo.schedule]
        cron = "0 0 * * * * *"
      [[job_store.local.job_generics.foo.commands]]
        command = "command1"
      [[job_store.local.job_generics.foo.commands]]
        command = "command2"
    [job_store.local.job_generics.linux_vm_job]
      [job_store.local.job_generics.linux_vm_job.command_generic]
        expect = ["vagrant@linux-vm:"]
		expect_not = ["No such file or directory"]
	[job_store.local.jobs.bar]
	  generic = "foo"
	[job_store.local.jobs.show_config]
	  generic = "linux_vm_job"
	  [job_store.local.jobs.show_config.schedule]
	    cron = "*/3 * * * * * *"
	  [[job_store.local.jobs.show_config.commands]]
	    command = "cat /vagrant/sample-txt"
	  [[job_store.local.jobs.show_config.scrubbers]]
	    type = "simple"
	    search = "multilink bundle-name authenticated"
	    replace = "multilink bundle-name {{replace_authenticated}}"
	  [job_store.local.jobs.show_config.template_values]
	    replace_authenticated = "job-level"

[device_store]
type = "local"
  [device_store.local]
    [device_store.local.device_generics]
      [device_store.local.device_generics.baz]
      [device_store.local.device_generics.linux_vm_base]
        host = "127.0.0.1"
        [device_store.local.device_generics.linux_vm_base.protocol]
          type = "ssh"
          [device_store.local.device_generics.linux_vm_base.protocol.ssh]
            port = 3222
	    [device_store.local.device_generics.linux_vm_base.credentials]
		  user = "vagrant"
		  pass = "vagrant"
    [device_store.local.devices.qux]
      host = "127.0.0.1"
      generic = "baz"
      [device_store.local.devices.qux.protocol]
        type = "ssh"
    [device_store.local.devices.local_linux_vm]
      generic = "linux_vm_base"
      [device_store.local.devices.local_linux_vm.jobs.show_config.template_values]
        replace_authenticated = "device-level"

[data_store]
type = "git"
  [data_store.git]
    url = "someurl1"
    pool_size = 1
    data_dir = "somedir1"
      [data_store.git.user]
        friendly_name = "John Doe"
        email = "jdoe@example.com"
	`
	assertValidConfig(t, toml, config.TOMLFormat)
}

func TestHCLUnmarshal(t *testing.T) {
	t.Skip("HCL currently does not work: https://github.com/hashicorp/hcl/issues/57")
	hcl := `
	ip = "127.0.0.1"
	port = 9400

	job_store {
	  type = "local"
	  local {
	    job_generics {
	  	  foo = {
	  	  	schedule { cron = "0 0 * * * * *" }
	  	  	commands a { command = "command1" }
	  	  	commands b { command = "command2" }
		  }
		  linux_vm_job {
		    command_generic {
		      expect = ["vagrant@linux-vm:"]
		      expect_not = ["No such file or directory"]
		    }
		  }
		}
		jobs {
		  bar { generic = "foo" }
		  show_config {
		    generic = "linux_vm_job"
		    schedule { cron = "*/3 * * * * * *" }
		    commands a { command = "cat /vagrant/sample-txt" }
		    scrubbers a {
		      type = "simple"
		      search = "multilink bundle-name authenticated"
		      replace = "multilink bundle-name {{replace_authenticated}}"
		    }
		    template_values { replace_authenticated = "job-level" }
		  }
		}
	  }
	}

	device_store {
	  type = "local"
	  local {
	    device_generics {
	      baz = {}
	      linux_vm_base {
	        host = "127.0.0.1"
	        protocol {
	          type = "ssh"
	          ssh { port = 3222 }
	        }
	        credentials {
	          user = "vagrant"
	          pass = "vagrant"
	        }
	      }
	    }
	    devices {
	      qux {
	        host = "127.0.0.1"
	        generic = "baz"
	        protocol { type = "ssh" }
	      }
	      local_linux_vm {
	        generic = "linux_vm_base"
	        jobs {
	          show_config {
	            template_values { replace_authenticated = "device-level" }
	          }
	        }
	      }
	    }
	  }
	}

	data_store {
	  type = "git"
	  git {
	    url = "someurl1"
	    pool_size = 1
	    data_dir = "somedir1"
	    user {
	      friendly_name = "John Doe"
	      email = "jdoe@example.com"
	    }
	  }
	}
	`
	assertValidConfig(t, hcl, config.HCLFormat)
}

func assertValidConfig(t *testing.T, contents string, format config.Format) {
	conf, err := config.NewFromBytes([]byte(contents), format)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(conf, simpleConfig) {
		actual, _ := conf.ToJSON(true)
		t.Fatalf("Not equal, actual:\n%v", string(actual))
	}
}

// Stolen from integration tests
var simpleConfig *config.Config = &config.Config{
	Ip:   "127.0.0.1",
	Port: 9400,
	JobStore: &config.JobStore{
		Type: "local",
		JobStoreLocal: &config.JobStoreLocal{
			JobGenerics: map[string]*config.Job{
				"foo": &config.Job{
					JobSchedule: &config.JobSchedule{
						Cron: "0 0 * * * * *",
					},
					Commands: []*config.JobCommand{
						&config.JobCommand{
							Command: "command1",
						},
						&config.JobCommand{
							Command: "command2",
						},
					},
				},
				"linux_vm_job": &config.Job{
					CommandGeneric: &config.JobCommand{
						Expect:    []string{"vagrant@linux-vm:"},
						ExpectNot: []string{"No such file or directory"},
					},
				},
			},
			Jobs: map[string]*config.Job{
				"bar": &config.Job{
					Generic: "foo",
				},
				"show_config": &config.Job{
					Generic:     "linux_vm_job",
					JobSchedule: &config.JobSchedule{Cron: "*/3 * * * * * *"},
					Commands: []*config.JobCommand{
						&config.JobCommand{Command: "cat /vagrant/sample-txt"},
					},
					Scrubbers: []*config.JobScrubber{
						&config.JobScrubber{
							Type:    "simple",
							Search:  "multilink bundle-name authenticated",
							Replace: "multilink bundle-name {{replace_authenticated}}",
						},
					},
					TemplateValues: map[string]string{
						"replace_authenticated": "job-level",
					},
				},
			},
		},
	},
	DeviceStore: &config.DeviceStore{
		Type: "local",
		DeviceStoreLocal: &config.DeviceStoreLocal{
			DeviceGenerics: map[string]*config.Device{
				"baz": &config.Device{},
				"linux_vm_base": &config.Device{
					Host: "127.0.0.1",
					DeviceProtocol: &config.DeviceProtocol{
						Type:              "ssh",
						DeviceProtocolSsh: &config.DeviceProtocolSsh{Port: 3222},
					},
					DeviceCredentials: &config.DeviceCredentials{User: "vagrant", Pass: "vagrant"},
				},
			},
			Devices: map[string]*config.Device{
				"qux": &config.Device{
					Host:    "127.0.0.1",
					Generic: "baz",
					DeviceProtocol: &config.DeviceProtocol{
						Type: "ssh",
					},
				},
				"local_linux_vm": &config.Device{
					Generic: "linux_vm_base",
					Jobs: map[string]*config.Job{
						"show_config": &config.Job{
							// Change the replace_authenticated template value
							TemplateValues: map[string]string{
								"replace_authenticated": "device-level",
							},
						},
					},
				},
			},
		},
	},
	DataStore: &config.DataStore{
		Type: "git",
		DataStoreGit: &config.DataStoreGit{
			Url:      "someurl1",
			PoolSize: 1,
			DataDir:  "somedir1",
			DataStoreGitUser: &config.DataStoreGitUser{
				FriendlyName: "John Doe",
				Email:        "jdoe@example.com",
			},
		},
	},
}
