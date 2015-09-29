package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal("Command required")
	}
	switch os.Args[1] {
	case "build-arista-vm":
		if err := buildAristaVm(os.Args[2:]...); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unrecognized command: %v", os.Args[1])
	}
}

func buildAristaVm(args ...string) error {
	basePath, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("Unable to get absolute path: %v", err)
	}

	// Create alternative ovf file with updated vEOS.vmdk path
	ovfContents, err := ioutil.ReadFile("packer-veos/virtualbox/source/vEOS.ovf")
	if err != nil {
		return fmt.Errorf("Unable to read OVF file: %v", err)
	}
	// We accept this can only executed one at a time per machine
	if err := ioutil.WriteFile("vEOS-ovf-temp.ovf", ovfContents, os.ModePerm); err != nil {
		return fmt.Errorf("Unable to write ovf file: %v", err)
	}
	defer os.Remove("vEOS-ovf-temp.ovf")

	// Load existing packer config based on OS
	fileName := "packer-veos/virtualbox/vEOS.json"
	if runtime.GOOS == "windows" {
		fileName = "packer-veos/virtualbox/vEOS-windows.json"
	}
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Failed to read %v: %v", fileName, err)
	}
	conf := map[string]interface{}{}
	if err := json.Unmarshal(fileBytes, &conf); err != nil {
		return fmt.Errorf("Unable to parse JSON in %v: %v", fileName, err)
	}

	// Fully qualify the source_path of the first builder
	firstBuilder := conf["builders"].([]interface{})[0].(map[string]interface{})
	firstBuilder["source_path"] = filepath.Join(basePath, "vEOS-ovf-temp.ovf")

	// Remove export opts
	delete(firstBuilder, "export_opts")

	// Inject the post processor
	err = injectPostProcessor(conf, map[string]interface{}{
		"type":   "vagrant",
		"output": filepath.Join(basePath, "arista-vm.box"),
	})
	if err != nil {
		return err
	}
	confFile, err := ioutil.TempFile(os.TempDir(), "vEOS-conf-temp")
	if err != nil {
		return fmt.Errorf("Unable to create new temp file: %v", err)
	}
	defer func() {
		confFile.Close()
		os.Remove(confFile.Name())
	}()
	fileBytes, err = json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to marshal JSON: %v", err)
	}

	// Need to change path to the Aboot ISO
	fileString := strings.Replace(string(fileBytes), "source/Aboot-vEOS.iso", "Aboot-vEOS.iso", -1)

	// Need to use #6 instead...
	fileString = strings.Replace(fileString, "VirtualBox Host-Only Ethernet Adapter #7",
		"VirtualBox Host-Only Ethernet Adapter #6", -1)

	// Reduce wait time...
	fileString = strings.Replace(fileString, "2m30s", "1m45s", -1)

	_, err = confFile.WriteString(fileString)
	if err != nil {
		return fmt.Errorf("Unable to write to %v: %v", confFile.Name(), err)
	}

	// Now run packer
	cmd := exec.Command("packer", "build", "-only", "vEOS1", confFile.Name())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Packer failed: %v", err)
	}
	return nil
}

func injectPostProcessor(conf map[string]interface{}, processor map[string]interface{}) error {
	if postProcessors, ok := conf["post-processors"]; !ok {
		conf["post-processors"] = []map[string]interface{}{processor}
	} else if postProcessorArray, ok := postProcessors.([]map[string]interface{}); !ok {
		return errors.New("Post processor config was not array")
	} else {
		conf["post-processors"] = append(postProcessorArray, processor)
	}
	return nil
}
