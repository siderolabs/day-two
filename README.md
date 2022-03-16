# day-two

This repo will hold templates and guides for our day 2 stack.
These will be things that we are willing to support directly as part of a support contract and will likely get deployed as part of an initial engagement.

This guide details manual deployment in-line here, but we also have a Pulumi script that does this.
You can simply run `d2ctl up` to deploy all the things.
This is intended to be a "fire and forget" type of tool, where we deploy these into a customer's Talos cluster as part of a PS engagement and later management is done by the customer directly via Helm.

## TL;DR

To get things installed, you should copy and edit `/hack/examples/config/config.yaml`, then run `d2ctl up --config-path /path/to/config.yaml`.
In that config file, you can specify each chart you want to deploy, comment out others, etc..
I've tried to include configs that work out of the box on QEMU-based Talos clusters.

## Building

You can build `d2ctl` with a quick `make d2ctl` from this repo.
The resulting binaries will be in `_out`.

## Monitoring and Logging

For monitoring and logging, we will recommend using a combination of Loki, Grafana, Prometheus, and Promtail.

The deployment of all of these tools can be done "all-in-one" with Helm via the Loki charts.

Chart location: [https://grafana.github.io/helm-charts](https://grafana.github.io/helm-charts) (specifically they loki-stack)

### Monitoring Notes and TODOs

- This will deploy with no persistent volume backing Prometheus.
  If we have access to PVs, we should set that value to true in the `helm upgrade` command.
- There are likely things like TLS that we'll need to configure here and should provide a `values.yaml` file once we've discovered that.

## Load Balancing

For load balancing, we'll recommend that clients use the "built-in" for their cloud platform if that's where they are running.

In the case of mixed envrionments or bare metal, we'll recommend MetalLB.

Chart location: [https://metallb.github.io/metallb](https://metallb.github.io/metallb)

### Load Balancing Notes

- The values.yaml file for MetalLB specifies a small IP pool.
  This should be updated depending on the network environment of the client.

## Ingress

For Ingress, we'll recommend the NGINX ingress controller, as it seems to be the most standard option in the community.

Chart location: [https://kubernetes.github.io/ingress-nginx](https://kubernetes.github.io/ingress-nginx)

By default, this will create an ingress service of type LoadBalancer (thus has some dependency on the LB section above) and should immediately be reachable via that IP/DNS name once everything is online.

### Ingress Notes and TODOs

- Document how to hook this into cert-manager

## SSL Certificates

For generating certs that can be used with Kubernetes applications, we recommend `cert-manager`.
Using cert-manager should let us hook into a variety of sources like Let's Encrypt, Vault, plus others.
We'll need to document each source as we encounter it and need to set it up.

Chart location: [https://charts.jetstack.io](https://charts.jetstack.io)

Once installed, you will need to create an Issuer or ClusterIssuer (ClusterIssuer if you don't want namespacing for the Issuer).
Here is a redacted example of using Route53 to solve the DNS challenges:

```bash
---
apiVersion: v1
kind: Secret
metadata:
  name: dev-route53-credentials-secret
  namespace: cert-manager
type: Opaque
data:
  secret-access-key: xxxxyyyyzzzzz
---
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: letsencrypt-dev
  namespace: cert-manager
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    # server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: aws+dev@talos-systems.io
    privateKeySecretRef:
      name: letsencrypt-dev
    solvers:
      # An empty 'selector' means that this solver matches all domains
      - selector: {}
        dns01:
          route53:
            accessKeyID: xxyyzz
            secretAccessKeySecretRef:
              name: dev-route53-credentials-secret
              key: secret-access-key
            region: us-west-2x
```

Lots more info [here](https://cert-manager.io/docs/configuration/acme/dns01/route53)

## Kube State Metrics

From their docs: "kube-state-metrics is a simple service that listens to the Kubernetes API server and generates metrics about the state of the objects.
It is not focused on the health of the individual Kubernetes components, but rather on the health of the various objects inside, such as deployments, nodes and pods."

This provides some basic etcd metrics and lots of k8s stuff so folks can setup alerts any way they want.
Note that this is just an easy starting point and we'll likely want to provider other/more metrics later on depending on client needs.

Chart location: [https://prometheus-community.github.io/helm-charts](https://prometheus-community.github.io/helm-charts)
