# Dex Server

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/dex-server)](https://artifacthub.io/packages/search?repo=dex-server)

Dex is a federated OpenID connect provider. It integrate any identity provider into your application using OpenID Connect. This server is used as a [LLMariner](https://github.com/llmariner/llmariner) sub-component. See [Technical Details](https://llmariner.ai/docs/dev/architecture/) document for details.

> [!NOTE]
> This is an alternative to the [Official Helm Chart](https://artifacthub.io/packages/helm/dex/dex) with the database integration support.

## Configuration

See [Customizing the Chart Before Installing](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing). To see all configurable options with detailed comments, visit the chart's [values.yaml](./values.yaml), or run these configuration commands:

```console
helm show values oci://public.ecr.aws/cloudnatix/llmariner-charts/dex-server
```

## Install Chart

```console
helm install <RELEASE_NAME> oci://public.ecr.aws/cloudnatix/llmariner-charts/dex-server
```

See [configuration](#configuration) below.
See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation.

## Uninstall Chart

```console
helm uninstall <RELEASE_NAME>
```

This removes all the Kubernetes components associated with the chart and deletes the release.
See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation.

## Upgrading Chart

```console
helm upgrade <RELEASE_NAME> oci://public.ecr.aws/cloudnatix/llmariner-charts/dex-server
```

See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation.
