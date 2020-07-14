package helmt

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var (
	Output       = color.Output
	Error        = color.Error
	Execute      = execCommand
	removeOutput = removeOutputCommand
	// use a single instance of Validate, it caches struct info
	validate *validator.Validate = validator.New()
)

type HelmChart struct {
	Chart      string   `yaml:"chart" validate:"required"`
	Version    string   `yaml:"version" validate:"required"`
	Repository string   `yaml:"repository" validate:"required"`
	Name       string   `yaml:"name" validate:"required"`
	Namespace  string   `yaml:"namespace"`
	Values     []string `yaml:"values"`
	skipCRDs   bool     `yaml:"skipCRDs"`
}

func readParameters(filename string) (*HelmChart, error) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	chart := &HelmChart{}
	err = yaml.Unmarshal(yamlFile, chart)
	if err != nil {
		return nil, err
	}

	err = validate.Struct(chart)
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func HelmTemplate(filename string, clean bool) error {
	chart, err := readParameters(filename)
	if err != nil {
		return err
	}

	err = HelmVersion()
	if err != nil {
		return err
	}

	err = fetch(chart.Repository, chart.Chart, chart.Version)
	if err != nil {
		return err
	}

	if clean {
		err = removeOutput(chart)
		if err != nil {
			return err
		}
	}

	chartVersion := fmt.Sprintf("%s-%s.tgz", chart.Chart, chart.Version)
	return template(chart.Name, chartVersion, chart.Values, chart.Namespace, chart.skipCRDs)
}

func HelmVersion() error {
	return Execute("helm", "version")
}

func template(name string, chart string, values []string, namespace string, skipCRDs bool) error {
	args := []string{"template", name, chart}
	if len(namespace) > 0 {
		args = append(args, "--namespace", namespace)
	}
	if !skipCRDs {
		args = append(args, "--include-crds")
	}
	for _, valuesfile := range values {
		args = append(args, "--values", valuesfile)
	}

	args = append(args, "--output-dir", ".")

	return Execute("helm", args...)
}

func fetch(repository, chart, version string) error {
	return Execute("helm", "fetch", fmt.Sprintf("%s/%s-%s.tgz", repository, chart, version))
}

func execCommand(name string, arg ...string) error {
	color.Magenta("%s %s", name, strings.Join(arg, " "))
	command := exec.Command(name, arg...)
	command.Stdout = Output
	command.Stderr = Error
	return command.Run()
}

func removeOutputCommand(chart *HelmChart) error {
	color.Magenta("removing folder %s", chart.Chart)
	return os.RemoveAll(chart.Chart)
}
