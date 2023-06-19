# lbaas-operator
There are use-cases that you may want to call an application outside the okd clusters.

Think of a scenario that you have 3 VMs that are running elasticsearch nodes and the application should call a URL that load balances requests to these three instances.

**lbaas-operator** provides a kind named `ExternalService` that simplifies the process of creating Services with custom Endpoints in OKD. It allows you to interact with external applications seamlessly.

An example for an `ExternalService` can be:

```yaml
apiVersion: networking.snappcloud.io/v1alpha1
kind: ExternalService
metadata:
  name: example-external-service
spec:
  type: static
  serviceType: ClusterIP
  static:
    addresses:
     - ip: 172.16.10.10
     - ip: 172.16.10.11
     - ip: 172.16.10.12
  ports:
    - name: 8080-tcp
      port: 8080
      protocol: TCP
```

### Specification

The `ExternalService` object has the following fields:

+ metadata: Metadata for the `ExternalService` resource, including the name and other optional fields.
+ spec: Specification for the `ExternalService`, including the type, serviceType, endpoints, and ports.
    + serviceType: The type of Kubernetes Service to create. For example, "ClusterIP" or "Loadbalancer".
    + ports: The list of ports to expose on the Service. Each port should specify the name, port number, and protocol.
    + type: The type of `ExternalService`. Currently, only the "static" type is supported.
    + static: Defines a static list of IP addresses that will be used as the endpoints for the Service.
        + addresses: A list of IP addresses that represent the endpoints for the Service.




### Development

* `make generate` update the generated code for that resource type.
* `make manifests` Generating CRD manifests.
* `make test` Run tests.

### Build

Export your image name:

```
export IMG=ghcr.io/snapp-incubator/lbaas-operator:main
```

* `make build` builds golang app locally.
* `make docker-build` build docker image locally.
* `make docker-push` push container image to registry.

### Run, Deploy
* `make run` run app locally
* `make deploy` deploy to k8s.

### Clean up

* `make undeploy` delete resouces in k8s.


## Security

### Reporting security vulnerabilities

If you find a security vulnerability or any security related issues, please DO NOT file a public issue, instead send your report privately to cloud@snapp.cab. Security reports are greatly appreciated and we will publicly thank you for it.

## License

Apache-2.0 License, see [LICENSE](LICENSE).