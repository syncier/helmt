package cmd

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		directory string
		files     []string
		wantErr   bool
	}{
		{
			name:    "helm-chart.yaml",
			args:    []string{"--version"},
			wantErr: false,
		},
		{
			name:    "jenkins",
			args:    []string{},
			files:   []string{"helm-chart.yaml"},
			wantErr: false,
		},
		{
			name: "prometheus",
			args: []string{"helm-chart-prometheus-operator.yaml"},
			files: []string{
				"helm-chart-prometheus-operator.yaml",
				"prometheus-operator-values.yaml",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			copyFileToDirectory(t, tt.files, dir)

			currentDir, _ := os.Getwd()
			defer func() { _ = os.Chdir(currentDir) }()
			err := os.Chdir(dir)
			assert.NoError(t, err)

			cmd := NewHelmtCommand("0.0.1-dummy")
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func copyFileToDirectory(t *testing.T, files []string, dir string) {
	for _, file := range files {
		data, err := ioutil.ReadFile(filepath.Join("testdata", file))
		assert.NoError(t, err)
		err = ioutil.WriteFile(filepath.Join(dir, file), data, 0644)
		assert.NoError(t, err)
	}
}

func TestGetFilename(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	t.Logf("Current test filename: %s", filename)
}
