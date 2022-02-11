# day-two
This repo will hold templates and guides for our day 2 stack. We'll see if it's worth keeping in here or if notion is good enough later.

This guide details manual deployment in-line here, but we also have a Pulumi script that does this.
You can simply run `go run main.go` to deploy all the things

TODO: everything is in default ns right now.

## Monitoring and Logging

For monitoring and logging, we will recommend using a combination of Loki, Grafana, Prometheus, and Promtail.

The deployment of all of these tools can be done "all-in-one" with Helm via the Loki charts.

To deploy:

```bash
helm repo add grafana https://grafana.github.io/helm-charts

helm repo update

helm upgrade --install loki --namespace loki grafana/loki-stack -f loki/values.yaml
```

### Notes and TODOs

- This will deploy with no persistent volume backing Prometheus. If we have access to PVs, we should set that value to true in the `helm upgrade` command.
- There are likely things like TLS that we'll need to configure here and should provide a `values.yaml` file once we've discovered that.

## Load Balancing

For load balancing, we'll recommend that clients use the "built-in" for their cloud platform if that's where they are running.

In the case of mixed envrionments or bare metal, we'll recommend MetalLB.

To deploy MetalLB:

```bash
helm repo add metallb https://metallb.github.io/metallb

helm repo update

helm upgrade --install metallb --namespace metallb metallb/metallb -f metallb/values.yaml
```

### Notes

- The values.yaml file for MetalLB specifies a small IP pool. This should be updated depending on the network environment of the client.

## Ingress

For Ingress, we'll recommend the NGINX ingress controller, as it seems to be the most standard option in the community. To deploy:

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx

helm repo update

helm upgrade --install ingress-nginx --namespace ingress ingress-nginx/ingress-nginx
```

By default, this will create an ingress service of type LoadBalancer and should immediately be reachable via that IP/DNS name once everything is online.

### Notes and TODOs

- Document how to hook this into cert-manager

## SSL Certificates

For generating certs that can be used with Kubernetes applications, we recommend `cert-manager`.
Using cert-manager should let us hook into a variety of sources like Let's Encrypt, Vault, plus others.
We'll need to document each source as we encounter it and need to set it up.

```bash
helm repo add cert-manager https://charts.jetstack.io

helm repo update

helm upgrade --install cert-manager --namespace cert-manager cert-manager/cert-manager -f cert-manager/values.yaml
```

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
