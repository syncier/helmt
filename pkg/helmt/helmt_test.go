package helmt

import (
	"github.com/otiai10/copy"
	"io/ioutil"
	"os"
	"path"
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

func (e *testExecutor) execCommand(name string, opts execOpts, arg ...string) error {
	e.commands = append(e.commands, strings.Join(append([]string{name}, arg...), " "))
	return nil
}

func (e *testExecutor) assertCommand(expected string) {
	assert.Equal(e.t, 1, len(e.commands))
	assert.Equal(e.t, e.commands[0], expected)
}

func TestHelmVersion(t *testing.T) {
	executor := NewTestExecutor(t)
	execute = executor.execCommand

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
				SkipCRDs:   true,
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
				Repository: "https://hub.syncier.cloud/chartrepo/library",
				Namespace:  "jenkins",
				Name:       "jenkins",
				Values:     []string{"values1.yaml", "values2.yaml"},
			},
		},
		{
			name: "generate kustomization",
			args: args{
				filename: "testdata/helm-chart-prometheus-operator.yaml",
			},
			want: &HelmChart{
				Chart:       "prometheus-operator",
				Version:     "8.12.15",
				Repository:  "https://kubernetes-charts.storage.googleapis.com",
				Namespace:   "infra-monitoring",
				Name:        "agent-prometheus",
				Values:      []string{"prometheus-operator-values.yaml"},
				PostProcess: PostProcess{GenerateKustomization: true},
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
		name                      string
		args                      args
		expectedCommands          []string
		wantRemoveOutput          bool
		wantGenerateKustomization bool
		wantErr                   bool
	}{
		{
			name: "helm template without values",
			args: args{
				filename: "testdata/helm-chart-mandatory-parameters.yaml",
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://kubernetes-charts.storage.googleapis.com --version 2.0.0 jenkins",
				"helm template something jenkins-2.0.0.tgz --include-crds --skip-tests --output-dir .",
			},
		},
		{
			name: "helm template with namespace and release",
			args: args{
				filename: "testdata/helm-chart.yaml",
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://hub.syncier.cloud/chartrepo/library --version 5.6.0 syncier-jenkins",
				"helm template jenkins syncier-jenkins-5.6.0.tgz --namespace jenkins --include-crds --skip-tests --values values1.yaml --values values2.yaml --output-dir .",
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
				"helm fetch --repo https://kubernetes-charts.storage.googleapis.com --version 2.0.0 jenkins",
				"helm template something jenkins-2.0.0.tgz --include-crds --skip-tests --output-dir .",
			},
			wantRemoveOutput: true,
		},
		{
			name: "skip crds",
			args: args{
				filename: "testdata/helm-chart-skip-crds.yaml",
				clean:    true,
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://kubernetes-charts.storage.googleapis.com --version 2.1.0 jenkins",
				"helm template something jenkins-2.1.0.tgz --skip-tests --output-dir .",
			},
			wantRemoveOutput: true,
		},
		{
			name: "generate kustomization",
			args: args{
				filename: "testdata/helm-chart-prometheus-operator.yaml",
				clean:    true,
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://kubernetes-charts.storage.googleapis.com --version 8.12.15 prometheus-operator",
				"helm template agent-prometheus prometheus-operator-8.12.15.tgz --namespace infra-monitoring --include-crds --skip-tests --values prometheus-operator-values.yaml --output-dir .",
			},
			wantRemoveOutput:          true,
			wantGenerateKustomization: true,
		},
		{
			name: "helm template outputDir in helm-chart.yaml",
			args: args{
				filename: "testdata/helm-chart-output-dir.yaml",
				clean:    true,
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://hub.syncier.cloud/chartrepo/library --version 5.6.0 syncier-jenkins",
				"helm template jenkins syncier-jenkins-5.6.0.tgz --namespace jenkins --include-crds --skip-tests --values values1.yaml --values values2.yaml --output-dir manifests",
			},
			wantRemoveOutput:          true,
			wantGenerateKustomization: false,
		},
		{
			name: "helm template with apiVersions in helm-chart.yaml",
			args: args{
				filename: "testdata/helm-chart-api-versions.yaml",
				clean:    true,
			},
			expectedCommands: []string{
				"helm version",
				"helm fetch --repo https://hub.syncier.cloud/chartrepo/library --version 5.6.0 syncier-jenkins",
				"helm template jenkins syncier-jenkins-5.6.0.tgz --namespace jenkins --include-crds --skip-tests --values values1.yaml --values values2.yaml --output-dir . --api-versions monitoring.coreos.com/v1 --api-versions monitoring.coreos.com/v1alpha1",
			},
			wantRemoveOutput:          true,
			wantGenerateKustomization: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor(t)
			execute = executor.execCommand

			outputRemoved := false
			removeOutput = func(_ *HelmChart) error {
				outputRemoved = true
				return nil
			}
			kustomizationGenerated := false
			generateKustomization = func(directory string) error {
				kustomizationGenerated = true
				return nil
			}

			if err := HelmTemplate(tt.args.filename, tt.args.clean); (err != nil) != tt.wantErr {
				t.Errorf("HelmTemplate() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.EqualValues(t, tt.expectedCommands, executor.commands)
				assert.Equal(t, tt.wantRemoveOutput, outputRemoved)
				assert.Equal(t, tt.wantGenerateKustomization, kustomizationGenerated)
			}
		})
	}
}

func Test_generateKustomization(t *testing.T) {
	type args struct {
		directory string
	}
	tests := []struct {
		name            string
		args            args
		expectedContent string
		wantErr         bool
	}{
		{
			name: "jenkins",
			args: args{
				directory: "testdata/jenkins",
			},
			expectedContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - templates/config.yaml
  - templates/home-pvc.yaml
  - templates/jcasc-config.yaml
  - templates/jenkins-agent-svc.yaml
  - templates/jenkins-master-deployment.yaml
  - templates/jenkins-master-svc.yaml
  - templates/rbac.yaml
  - templates/secret.yaml
  - templates/service-account.yaml
  - templates/tests/test-config.yaml`,
			wantErr: false,
		},
		{
			name: "prometheus-operator",
			args: args{
				directory: "testdata/prometheus-operator",
			},
			expectedContent: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - charts/grafana/templates/clusterrole.yaml
  - charts/grafana/templates/clusterrolebinding.yaml
  - charts/grafana/templates/configmap-dashboard-provider.yaml
  - charts/grafana/templates/configmap.yaml
  - charts/grafana/templates/deployment.yaml
  - charts/grafana/templates/ingress.yaml
  - charts/grafana/templates/podsecuritypolicy.yaml
  - charts/grafana/templates/role.yaml
  - charts/grafana/templates/rolebinding.yaml
  - charts/grafana/templates/service.yaml
  - charts/grafana/templates/serviceaccount.yaml
  - charts/grafana/templates/tests/test-configmap.yaml
  - charts/grafana/templates/tests/test-podsecuritypolicy.yaml
  - charts/grafana/templates/tests/test-role.yaml
  - charts/grafana/templates/tests/test-rolebinding.yaml
  - charts/grafana/templates/tests/test-serviceaccount.yaml
  - charts/kube-state-metrics/templates/clusterrole.yaml
  - charts/kube-state-metrics/templates/clusterrolebinding.yaml
  - charts/kube-state-metrics/templates/deployment.yaml
  - charts/kube-state-metrics/templates/podsecuritypolicy.yaml
  - charts/kube-state-metrics/templates/service.yaml
  - charts/kube-state-metrics/templates/serviceaccount.yaml
  - charts/prometheus-node-exporter/templates/daemonset.yaml
  - charts/prometheus-node-exporter/templates/psp-clusterrole.yaml
  - charts/prometheus-node-exporter/templates/psp-clusterrolebinding.yaml
  - charts/prometheus-node-exporter/templates/psp.yaml
  - charts/prometheus-node-exporter/templates/service.yaml
  - charts/prometheus-node-exporter/templates/serviceaccount.yaml
  - crds/crd-alertmanager.yaml
  - crds/crd-podmonitor.yaml
  - crds/crd-prometheus.yaml
  - crds/crd-prometheusrules.yaml
  - crds/crd-servicemonitor.yaml
  - crds/crd-thanosrulers.yaml
  - templates/alertmanager/alertmanager.yaml
  - templates/alertmanager/ingress.yaml
  - templates/alertmanager/psp-role.yaml
  - templates/alertmanager/psp-rolebinding.yaml
  - templates/alertmanager/psp.yaml
  - templates/alertmanager/secret.yaml
  - templates/alertmanager/service.yaml
  - templates/alertmanager/serviceaccount.yaml
  - templates/alertmanager/servicemonitor.yaml
  - templates/exporters/core-dns/service.yaml
  - templates/exporters/core-dns/servicemonitor.yaml
  - templates/exporters/kube-api-server/servicemonitor.yaml
  - templates/exporters/kube-controller-manager/service.yaml
  - templates/exporters/kube-controller-manager/servicemonitor.yaml
  - templates/exporters/kube-etcd/service.yaml
  - templates/exporters/kube-etcd/servicemonitor.yaml
  - templates/exporters/kube-proxy/service.yaml
  - templates/exporters/kube-proxy/servicemonitor.yaml
  - templates/exporters/kube-scheduler/service.yaml
  - templates/exporters/kube-scheduler/servicemonitor.yaml
  - templates/exporters/kube-state-metrics/serviceMonitor.yaml
  - templates/exporters/kubelet/servicemonitor.yaml
  - templates/exporters/node-exporter/servicemonitor.yaml
  - templates/grafana/configmaps-datasources.yaml
  - templates/grafana/dashboards-1.14/apiserver.yaml
  - templates/grafana/dashboards-1.14/cluster-total.yaml
  - templates/grafana/dashboards-1.14/controller-manager.yaml
  - templates/grafana/dashboards-1.14/etcd.yaml
  - templates/grafana/dashboards-1.14/k8s-coredns.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-cluster.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-namespace.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-node.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-pod.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-workload.yaml
  - templates/grafana/dashboards-1.14/k8s-resources-workloads-namespace.yaml
  - templates/grafana/dashboards-1.14/kubelet.yaml
  - templates/grafana/dashboards-1.14/namespace-by-pod.yaml
  - templates/grafana/dashboards-1.14/namespace-by-workload.yaml
  - templates/grafana/dashboards-1.14/node-cluster-rsrc-use.yaml
  - templates/grafana/dashboards-1.14/node-rsrc-use.yaml
  - templates/grafana/dashboards-1.14/nodes.yaml
  - templates/grafana/dashboards-1.14/persistentvolumesusage.yaml
  - templates/grafana/dashboards-1.14/pod-total.yaml
  - templates/grafana/dashboards-1.14/pods.yaml
  - templates/grafana/dashboards-1.14/prometheus.yaml
  - templates/grafana/dashboards-1.14/proxy.yaml
  - templates/grafana/dashboards-1.14/scheduler.yaml
  - templates/grafana/dashboards-1.14/statefulset.yaml
  - templates/grafana/dashboards-1.14/workload-total.yaml
  - templates/grafana/servicemonitor.yaml
  - templates/prometheus/additionalPrometheusRules.yaml
  - templates/prometheus/clusterrole.yaml
  - templates/prometheus/clusterrolebinding.yaml
  - templates/prometheus/ingress.yaml
  - templates/prometheus/prometheus.yaml
  - templates/prometheus/psp-clusterrole.yaml
  - templates/prometheus/psp-clusterrolebinding.yaml
  - templates/prometheus/psp.yaml
  - templates/prometheus/rules-1.14/alertmanager.rules.yaml
  - templates/prometheus/rules-1.14/etcd.yaml
  - templates/prometheus/rules-1.14/general.rules.yaml
  - templates/prometheus/rules-1.14/k8s.rules.yaml
  - templates/prometheus/rules-1.14/kube-apiserver-error-alerts.yaml
  - templates/prometheus/rules-1.14/kube-apiserver-error.yaml
  - templates/prometheus/rules-1.14/kube-apiserver-slos.yaml
  - templates/prometheus/rules-1.14/kube-apiserver.rules.yaml
  - templates/prometheus/rules-1.14/kube-prometheus-general.rules.yaml
  - templates/prometheus/rules-1.14/kube-prometheus-node-recording.rules.yaml
  - templates/prometheus/rules-1.14/kube-scheduler.rules.yaml
  - templates/prometheus/rules-1.14/kube-state-metrics.yaml
  - templates/prometheus/rules-1.14/kubelet.rules.yaml
  - templates/prometheus/rules-1.14/kubernetes-absent.yaml
  - templates/prometheus/rules-1.14/kubernetes-apps.yaml
  - templates/prometheus/rules-1.14/kubernetes-resources.yaml
  - templates/prometheus/rules-1.14/kubernetes-storage.yaml
  - templates/prometheus/rules-1.14/kubernetes-system-apiserver.yaml
  - templates/prometheus/rules-1.14/kubernetes-system-controller-manager.yaml
  - templates/prometheus/rules-1.14/kubernetes-system-kubelet.yaml
  - templates/prometheus/rules-1.14/kubernetes-system-scheduler.yaml
  - templates/prometheus/rules-1.14/kubernetes-system.yaml
  - templates/prometheus/rules-1.14/node-exporter.rules.yaml
  - templates/prometheus/rules-1.14/node-exporter.yaml
  - templates/prometheus/rules-1.14/node-network.yaml
  - templates/prometheus/rules-1.14/node-time.yaml
  - templates/prometheus/rules-1.14/node.rules.yaml
  - templates/prometheus/rules-1.14/prometheus-operator.yaml
  - templates/prometheus/rules-1.14/prometheus.yaml
  - templates/prometheus/service.yaml
  - templates/prometheus/serviceaccount.yaml
  - templates/prometheus/servicemonitor.yaml
  - templates/prometheus-operator/admission-webhooks/mutatingWebhookConfiguration.yaml
  - templates/prometheus-operator/admission-webhooks/validatingWebhookConfiguration.yaml
  - templates/prometheus-operator/clusterrole.yaml
  - templates/prometheus-operator/clusterrolebinding.yaml
  - templates/prometheus-operator/deployment.yaml
  - templates/prometheus-operator/psp-clusterrole.yaml
  - templates/prometheus-operator/psp-clusterrolebinding.yaml
  - templates/prometheus-operator/psp.yaml
  - templates/prometheus-operator/service.yaml
  - templates/prometheus-operator/serviceaccount.yaml
  - templates/prometheus-operator/servicemonitor.yaml`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "kustomization")
			assert.NoError(t, err)
			err = copy.Copy(tt.args.directory, dir)
			assert.NoError(t, err)

			if err := generateKustomizationCommand(dir); (err != nil) != tt.wantErr {
				t.Errorf("generateKustomizationCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			kustomization := path.Join(dir, "kustomization.yaml")
			stat, err := os.Stat(kustomization)
			if err != nil {
				assert.FailNow(t, "kustomization.yaml was not generated")
			}

			assert.False(t, stat.IsDir())
			assert.Equal(t, tt.expectedContent, ReadFileAsString(t, kustomization))
		})
	}
}

func ReadFileAsString(t *testing.T, filename string) string {
	expectedContent, err := ioutil.ReadFile(filename)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	return strings.TrimSuffix(string(expectedContent), "\n")
}
