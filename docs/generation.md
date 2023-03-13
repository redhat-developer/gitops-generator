# GitOps Resource Generation Logic

## Base Resources

The `Generate()` function in [generate.go](../pkg/generate.go) is responsible for generating the base resources in the GitOps repository. The `GeneratorOptions` struct in [generator_options.go](../api/v1alpha1/generator_options.go) specifies the required configuration for the generator options.

### Bringing your own resource(s)

Specify your own Deployments, Services and Routes via `GeneratorOptions.KubernetesResources`. These resources will be written to the GitOps repository. Other resource types should be specified in `GeneratorOptions.KubernetesResources.Others`.

If more than one Deployment, Service or Route is specified; then the first corresponding resource of each type are written into their specific file, while the remaining resources are appended to Others. For instance if there are two Deployments, the first Deployment is written to the file `deployment.yaml` and the second Deployment is appended to the file `other_resources.yaml`.

### Not bringing your own resource(s)

If `GeneratorOptions.KubernetesResources` are not specified, then a Deployment, Service and Route will be generated based on the information provided in `GeneratorOptions`. For a Service and Route to be generated, the `GeneratorOptions.TargetPort` should not be 0.

## Overlays Resources

The `GenerateOverlays()` function in [generate.go](../pkg/generate.go) is responsible for generating the overlays resources in the GitOps repository. The `GeneratorOptions` struct in [generator_options.go](../api/v1alpha1/generator_options.go) specifies the required configuration for the generator options.

The `GenerateOverlays()` function also preserves any custom patching that was introduced in `kustomization.yaml`.