package helmt

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var (
	Output                = color.Output
	Error                 = color.Error
	execute               = execCommand
	removeOutput          = removeOutputCommand
	generateKustomization = generateKustomizationCommand
	// use a single instance of Validate, it caches struct info
	validate *validator.Validate = validator.New()
)

type HelmChart struct {
	Chart       string      `yaml:"chart" validate:"required"`
	Version     string      `yaml:"version" validate:"required"`
	Repository  string      `yaml:"repository" validate:"required"`
	Name        string      `yaml:"name" validate:"required"`
	Namespace   string      `yaml:"namespace"`
	Values      []string    `yaml:"values"`
	SkipCRDs    bool        `yaml:"skipCRDs"`
	PostProcess PostProcess `yaml:"postProcess"`
	OutputDir   string      `yaml:"outputDir"`
	apiVersions []string    `yaml:"apiVersions"`
}

type PostProcess struct {
	GenerateKustomization bool `yaml:"generateKustomization"`
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
	err = template(chart.Name, chartVersion, chart.Values, chart.Namespace, chart.SkipCRDs, chart.OutputDir, chart.apiVersions)
	if err != nil {
		return err
	}

	if chart.PostProcess.GenerateKustomization {
		err = generateKustomization(chart.Chart)
		if err != nil {
			return err
		}
	}

	err = downloadChartMetadata(chart.Chart, chart.Repository, chart.Version, chart.OutputDir)
	if err != nil {
		println(fmt.Sprintf("Warning: Could not retrieve Chart.yaml (%v)", err))
	}

	return nil
}

func HelmVersion() error {
	return execute("helm", execOpts{}, "version")
}

func template(name string, chart string, values []string, namespace string, skipCRDs bool, outputDir string, apiVersions []string) error {
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
	if len(outputDir) > 0 {
		args = append(args, "--output-dir", outputDir)
	} else {
		args = append(args, "--output-dir", ".")
	}
	for _, apiversion := range apiVersions {
		args = append(args, "--api-versions", apiversion)
	}

	return execute("helm", execOpts{}, args...)
}

func fetch(repository, chart, version string) error {
	return execute("helm", execOpts{}, "fetch", "--repo", repository, "--version", version, chart)
}

func downloadChartMetadata(chart string, repo string, version string, outputDir string) error {
	args := []string{"show", "chart", chart, "--repo", repo, "--version", version}

	output := &bytes.Buffer{}
	// Due to https://github.com/helm/helm/issues/6864 we have to run the command in another directory.
	// Otherwise the local directory with the chart name will be taken by helm, instead of downloading from the remote repo.
	err := execute("helm", execOpts{Dir: os.TempDir(), Output: output}, args...)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(chart, "Chart.yaml")
	if len(outputDir) > 0 {
		targetPath = filepath.Join(outputDir, targetPath)
	}

	return ioutil.WriteFile(targetPath, output.Bytes(), os.ModePerm)
}

type execOpts struct {
	Dir    string
	Output io.Writer
}

func execCommand(name string, opts execOpts, arg ...string) error {
	color.Magenta("%s %s", name, strings.Join(arg, " "))
	command := exec.Command(name, arg...)
	command.Dir = opts.Dir
	if opts.Output != nil {
		command.Stdout = opts.Output
	} else {
		command.Stdout = Output
	}
	command.Stderr = Error
	return command.Run()
}

func removeOutputCommand(chart *HelmChart) error {
	color.Magenta("removing folder %s", chart.Chart)
	return os.RemoveAll(chart.Chart)
}

func generateKustomizationCommand(directory string) error {
	kustomization, err := os.Create(path.Join(directory, "kustomization.yaml"))
	if err != nil {
		return err
	}
	defer kustomization.Close()

	_, err = kustomization.WriteString(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`)
	if err != nil {
		return err
	}

	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		if rel == "kustomization.yaml" {
			return nil
		}

		_, err = kustomization.WriteString(fmt.Sprintf("  - %s\n", rel))
		if err != nil {
			return err
		}
		return nil
	})

	return err
}
