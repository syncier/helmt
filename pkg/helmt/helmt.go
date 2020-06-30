package helmt

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var (
	Output  = color.Output
	Error   = color.Error
	Execute = execCommand
	// use a single instance of Validate, it caches struct info
	validate *validator.Validate = validator.New()
)

type HelmChart struct {
	Chart      string   `yaml:"chart" validate:"required"`
	Version    string   `yaml:"version" validate:"required"`
	Repository string   `yaml:"repository" validate:"required"`
	Namespace  string   `yaml:"namespace"`
	Values     []string `yaml:"values"`
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

func HelmTemplate(filename string) error {
	chart, err := readParameters(filename)
	if err != nil {
		return err
	}
	err = fetch(chart.Repository, chart.Chart, chart.Version)
	if err != nil {
		return err
	}

	chartVersion := fmt.Sprintf("%s-%s.tgz", chart.Chart, chart.Version)
	return template(chartVersion, chart.Values, chart.Namespace)
}

func HelmVersion() error {
	return Execute("helm", "version")
}

func template(chartVersion string, values []string, namespace string) error {
	//return Execute("helm", "template", "--namespace", namespace, "--values", values[0], "--output-dir", ".", chartVersion)
	args := []string{"template"}
	if len(namespace) > 0 {
		args = append(args, "--namespace", namespace)
	}
	for _, valuesfile := range values {
		args = append(args, "--values", valuesfile)
	}

	args = append(args, "--output-dir", ".", chartVersion)

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