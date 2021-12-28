# helmt - A simple wrapper around helm template

Sometimes you only want to use [helm](https://helm.sh/) to template Kubernetes manifests.

```A simple wrapper around helm template
Usage:
  helmt <filename> [flags]

Flags:
      --config string     config file (default is $HOME/.helmt.yaml)
  -h, --help              help for helmt
  -p, --password string   optional password for chart repository
  -u, --username string   optional username for chart repository
  -v, --version           version for helmt
```

## Flags, environment variables and config file

The following flags can also be set via environment variables.
But command line parameters have always precedence.

| Flag     | Environment variable |
| -------- | -------------------- |
| config   | `HELMT_CONFIG`       |
| username | `HELMT_USERNAME`     |
| password | `HELMT_PASSWORD`     |

The config is a simple yaml file with the names of the flags as keys.
Example:

```yaml
username: anonymous
```

Please note that environment variables and command line parameters overwrite these settings.

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
helm template jenkins jenkins-2.0.0.tgz --output-dir .
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

If the chart repository needs authentication, provide credentials via `--username` and `--password` or environment variables.
