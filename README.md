# gitops-generator

[![codecov](https://codecov.io/gh/redhat-developer/gitops-generator/branch/main/graph/badge.svg)](https://codecov.io/gh/redhat-developer/gitops-generator)

A community open-source library to help with your project's GitOps needs.

This library generates the Kubernetes resource files and uses the Kustomize tool. The files are then pushed to the Git repository specified.

For more information on the specifics of the resources generated, please refer to the [generation](./docs/generation.md) documentation

## Developement & Testing

### Prerequisite
- go 1.18 or later

### Testing

To run all the tests in this repository

```
make test
```

## Contributions

Please see our [CONTRIBUTING](./CONTRIBUTING.md) for more information.
