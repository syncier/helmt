package helmt

import (
	"reflect"
	"strings"
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func NewTestExecutor(t *testing.T) *testExecutor {
	return &testExecutor{t: t, commands: []string{}}
}

type testExecutor struct {
	commands []string
	t        *testing.T
}

func (e *testExecutor) execCommand(name string, arg ...string) error {
	e.commands = append(e.commands, strings.Join(append([]string{name}, arg...), " "))
	return nil
}

func (e *testExecutor) assertCommand(expected string) {
	assert.Equal(e.t, 1, len(e.commands))
	assert.Equal(e.t, e.commands[0], expected)
}

func TestHelmVersion(t *testing.T) {
	executor := NewTestExecutor(t)
	Execute = executor.execCommand

	err := HelmVersion()
	assert.NoError(t, err)

	executor.assertCommand("helm version")
}

func Test_readParameters(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    *HelmChart
		wantErr bool
	}{
		{
			name: "non existing file",
			args: args{
				filename: "tstdata/does-not-exist.yaml",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty file",
			args: args{
				filename: "tstdata/empty.yaml",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing repo",
			args: args{
				filename: "testdata/helm-chart-missing-repo.yaml",
			},
			wantErr: true,
		},
		{
			name: "missing chart",
			args: args{
				filename: "testdata/helm-chart-missing-chart.yaml",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			args: args{
				filename: "testdata/helm-chart-missing-version.yaml",
			},
			wantErr: true,
		},
		{
			name: "missing release",
			args: args{
				filename: "testdata/helm-chart-missing-release.yaml",
			},
			wantErr: true,
		},
		{
			name: "file with mandatory parameters only",
			args: args{
				filename: "testdata/helm-chart-mandatory-parameters.yaml",
			},
			want: &HelmChart{
				Chart:      "jenkins",
				Version:    "2.0.0",
				Repository: "https://kubernetes-charts.storage.googleapis.com",
				Name:       "something",
			},
		},
		{
			name: "empty namespace",
			args: args{
				filename: "testdata/helm-chart-empty-namespace.yaml",
			},
			want: &HelmChart{
				Chart:      "stable/jenkins",
				Version:    "2.0.0",
				Repository: "https://kubernetes-charts.storage.googleapis.com",
				Name:       "my-jenkins",
			},
		},
		{
			name: "file with all parameters",
			args: args{
				filename: "testdata/helm-chart.yaml",
			},
			want: &HelmChart{
				Chart:      "syncier-jenkins",
				Version:    "5.6.0",
				Repository: "https://hub.syncier.cloud/chartrepo/library/charts",
				Namespace:  "jenkins",
				Name:       "jenkins",
				Values:     []string{"values1.yaml", "values2.yaml"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readParameters(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("readParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readParameters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmTemplate(t *testing.T) {
	type args struct {
		filename string
		clean    bool
	}
	tests := []struct {
		name             string
		args             args
		expectedCommands []string
		wantRemoveOutput bool
		wantErr          bool
	}{
		{
			name: "helm template without values",
			args: args{
				filename: "testdata/helm-chart-mandatory-parameters.yaml",
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch https://kubernetes-charts.storage.googleapis.com/jenkins-2.0.0.tgz",
				"helm template something --output-dir . jenkins-2.0.0.tgz",
			},
		},
		{
			name: "helm template with namespace and release",
			args: args{
				filename: "testdata/helm-chart.yaml",
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch https://hub.syncier.cloud/chartrepo/library/charts/syncier-jenkins-5.6.0.tgz",
				"helm template jenkins --namespace jenkins --values values1.yaml --values values2.yaml --output-dir . syncier-jenkins-5.6.0.tgz",
			},
		},
		{
			name: "helm template with namespace and release",
			args: args{
				filename: "testdata/helm-chart-missing-release.yaml",
			},
			expectedCommands: []string{},
			wantErr:          true,
		},
		{
			name: "with clean flag",
			args: args{
				filename: "testdata/helm-chart-mandatory-parameters.yaml",
				clean:    true,
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch https://kubernetes-charts.storage.googleapis.com/jenkins-2.0.0.tgz",
				"helm template something --output-dir . jenkins-2.0.0.tgz",
			},
			wantRemoveOutput: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor(t)
			Execute = executor.execCommand

			outputRemoved := false
			removeOutput = func(_ *HelmChart) error {
				outputRemoved = true
				return nil
			}

			if err := HelmTemplate(tt.args.filename, tt.args.clean); (err != nil) != tt.wantErr {
				t.Errorf("HelmTemplate() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.EqualValues(t, tt.expectedCommands, executor.commands)
				assert.Equal(t, tt.wantRemoveOutput, outputRemoved)
			}
		})
	}
}
