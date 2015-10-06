package model

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
)

type FileSet struct {
	Files []*FileSetFile
}

func NewDefaultFileSet() *FileSet {
	return &FileSet{Files: []*FileSetFile{}}
}

func (f *FileSet) ApplyConfig(conf *config.Job) {
	for fileName, fileConf := range conf.JobFile {
		file := NewDefaultFileSetFile()
		file.ApplyConfig(fileName, fileConf)
		f.Files = append(f.Files, file)
	}
}

func (f *FileSet) Validate() []error {
	errs := []error{}
	if len(f.Files) == 0 {
		errs = append(errs, errors.New("No files in set"))
	}
	for _, file := range f.Files {
		for _, err := range file.Validate() {
			errs = append(errs, fmt.Errorf("File '%v' invalid: %v", file.Name, err))
		}
	}
	return errs
}

func (f *FileSet) DeepCopy() *FileSet {
	ret := &FileSet{Files: []*FileSetFile{}}
	for _, file := range f.Files {
		ret.Files = append(ret.Files, file.DeepCopy())
	}
	return ret
}

type FileSetFile struct {
	Name        string `json:"name"`
	Compression string `json:"compression"`
}

func NewDefaultFileSetFile() *FileSetFile {
	return &FileSetFile{}
}

func (f *FileSetFile) ApplyConfig(fileName string, conf *config.JobFile) {
	if fileName != "" {
		f.Name = fileName
	}
	if conf.Compression != "" {
		f.Compression = conf.Compression
	}
}

func (f *FileSetFile) Validate() []error {
	errs := []error{}
	if len(f.Name) == 0 {
		errs = append(errs, errors.New("No files in set"))
	}
	if f.Compression != "" && f.Compression != "gzip" {
		errs = append(errs, fmt.Errorf("Unrecognized compression: %v", f.Compression))
	}
	return errs
}

func (f *FileSetFile) DeepCopy() *FileSetFile {
	return &FileSetFile{Name: f.Name, Compression: f.Compression}
}
