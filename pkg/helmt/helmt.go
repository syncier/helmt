package helmt

import (
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
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	Output                = color.Output
	Error                 = color.Error
	execute               = execCommand
	generateKustomization = generateKustomizationCommand
	TempDir               = afero.TempDir
	// use a single instance of Validate, it caches struct info
	validate *validator.Validate = validator.New()
	fs                           = afero.NewOsFs()
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
	ApiVersions []string    `yaml:"apiVersions"`
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

func HelmTemplate(filename, username, password string) error {
	tmpDir, err := TempDir(fs, ".", "helmt")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer func() { _ = fs.RemoveAll(tmpDir) }()

	chart, err := readParameters(filename)
	if err != nil {
		return err
	}

	err = HelmVersion()
	if err != nil {
		return err
	}

	chartFile, err := fetch(tmpDir, chart.Repository, chart.Chart, chart.Version, username, password)
	if err != nil {
		return err
	}

	err = template(tmpDir, chart.Name, filepath.Join(tmpDir, chartFile), chart.Values, chart.Namespace, chart.SkipCRDs, chart.ApiVersions)
	if err != nil {
		return err
	}

	rendered := filepath.Join(tmpDir, chart.Chart)

	if chart.PostProcess.GenerateKustomization {
		err = generateKustomization(rendered)
		if err != nil {
			return err
		}
	}

	target := "."
	if chart.OutputDir != "" {
		target = chart.OutputDir
	}
	target = filepath.Join(target, chart.Chart)

	err = fs.RemoveAll(target)
	if err != nil {
		return err
	}

	err = fs.Rename(rendered, target)
	if err != nil {
		return fmt.Errorf("failed to move rendered chart: %v", err)
	}

	return nil
}

func HelmVersion() error {
	return execute("helm", execOpts{}, "version")
}

func template(tmpDir string, name string, chart string, values []string, namespace string, skipCRDs bool, ApiVersions []string) error {
	args := []string{"template", name, chart}
	if len(namespace) > 0 {
		args = append(args, "--namespace", namespace)
	}
	if !skipCRDs {
		args = append(args, "--include-crds")
	}
	args = append(args, "--skip-tests")
	for _, valuesfile := range values {
		args = append(args, "--values", valuesfile)
	}
	args = append(args, "--output-dir", tmpDir)
	if len(ApiVersions) > 0 {
		for _, apiversion := range ApiVersions {
			args = append(args, "--api-versions", apiversion)
		}
	}

	err := execute("helm", execOpts{}, args...)
	if err != nil {
		return fmt.Errorf("helm template failed: %v", err)
	}
	return nil
}

func fetch(tmpDir, repository, chart, version string, username, password string) (string, error) {
	isOCI := strings.HasPrefix(repository, "oci://")
	if isOCI {
		repository = strings.Join([]string{repository, chart}, "/")
	}

	args := []string{"fetch"}
	if !isOCI {
		args = append(args, "--repo")
	}
	args = append(args, repository)
	args = append(args, "--version", version)
	args = append(args, "--destination", tmpDir)
	if username != "" {
		args = append(args, "--username", username)
	}
	if password != "" {
		args = append(args, "--password", password)
	}

	if !isOCI {
		args = append(args, chart)
	}
	err := execute("helm", execOpts{}, args...)
	if err != nil {
		return "", err
	}

	result, err := findChartPackage(tmpDir)
	if err != nil {
		return "", err
	}
	color.Magenta("downloaded %s", result)
	return result, nil
}

type execOpts struct {
	Dir    string
	Output io.Writer
}

func execCommand(name string, opts execOpts, arg ...string) error {
	args := strings.Join(arg, " ")
	args = strings.ReplaceAll(args, "--password "+viper.GetString("password"), "--password *****")
	color.Magenta("%s %s", name, args)

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

func generateKustomizationCommand(directory string) error {
	kustomization, err := fs.Create(path.Join(directory, "kustomization.yaml"))
	if err != nil {
		return err
	}
	defer func() { _ = kustomization.Close() }()

	_, err = kustomization.WriteString(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`)
	if err != nil {
		return err
	}

	err = afero.Walk(fs, directory, func(path string, info os.FileInfo, err error) error {
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

func findChartPackage(dir string) (string, error) {
	c, err := afero.ReadDir(fs, dir)
	if err != nil {
		return "", err
	}
	for _, f := range c {
		if filepath.Ext(f.Name()) == ".tgz" {
			return f.Name(), nil
		}
	}
	return "", fmt.Errorf("unexpected content in temporary directory %v", dir)
}
