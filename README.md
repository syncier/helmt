# helmt - A simple wrapper around helm template

Sometimes you only want to use [helm](https://helm.sh/) to template Kubernetes manifests.

```A simple wrapper around helm template
Usage:
  helmt <filename> [flags]

Flags:
      --config string   config file (default is $HOME/.helmt.yaml)
  -h, --help            help for helmt
  -v, --version         version for helmt
```

## Example

All you need to do is to create a yaml file describing which chart you want to template:

```yaml
chart: jenkins
version: 2.0.0
repository: https://kubernetes-charts.storage.googleapis.com
name: jenkins
```

or

```yaml
chart: syncier-jenkins
version: 5.6.0
repository: https://hub.syncier.cloud/chartrepo/library/charts
namespace: jenkins
name: jenkins
values:
  - values1.yaml
  - values2.yaml
```

Then you can run `helmt helm-charts.yaml` and it will download the chart and render the contents using the parameters defined in the yaml file.

```shell script
helm version
version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}
helm fetch https://kubernetes-charts.storage.googleapis.com/jenkins-2.0.0.tgz
helm template jenkins --output-dir . jenkins-2.0.0.tgz
wrote ./jenkins/templates/service-account.yaml
wrote ./jenkins/templates/secret.yaml
wrote ./jenkins/templates/config.yaml
wrote ./jenkins/templates/jcasc-config.yaml
wrote ./jenkins/templates/tests/test-config.yaml
wrote ./jenkins/templates/home-pvc.yaml
wrote ./jenkins/templates/rbac.yaml
wrote ./jenkins/templates/rbac.yaml
wrote ./jenkins/templates/rbac.yaml
wrote ./jenkins/templates/rbac.yaml
wrote ./jenkins/templates/jenkins-agent-svc.yaml
wrote ./jenkins/templates/jenkins-master-svc.yaml
wrote ./jenkins/templates/jenkins-master-deployment.yaml
```
