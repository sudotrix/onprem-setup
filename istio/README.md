# Istio

An open platform to connect, manage, and secure microservices.
- For in-depth information about how to use Istio, visit [istio.io](https://istio.io)


## Introduction

Istio is an open platform for providing a uniform way to integrate
microservices, manage traffic flow across microservices, enforce policies
and aggregate telemetry data. Istio's control plane provides an abstraction
layer over the underlying cluster management platform, such as Kubernetes.

Istio is composed of these components:

- **Envoy** - Sidecar proxies per microservice to handle ingress/egress traffic
   between services in the cluster and from a service to external
   services. The proxies form a _secure microservice mesh_ providing a rich
   set of functions like discovery, rich layer-7 routing, circuit breakers,
   policy enforcement and telemetry recording/reporting
   functions.

  > Note: The service mesh is not an overlay network. It
  > simplifies and enhances how microservices in an application talk to each
  > other over the network provided by the underlying platform.

- **Mixer** - Central component that is leveraged by the proxies and microservices
   to enforce policies such as authorization, rate limits, quotas, authentication, request
   tracing and telemetry collection.

- **Pilot** - A component responsible for configuring the proxies at runtime.

- **Citadel** - A centralized component responsible for certificate issuance and rotation.

- **Citadel Agent** - A per-node component responsible for certificate issuance and rotation.

- **Galley**- Central component for validating, ingesting, aggregating, transforming and distributing config within Istio.

Istio currently supports Kubernetes and Consul-based environments. 



# To set up istio in the new k8s cluster follow bellow steps 
- **Download istio binaries**

wget https://github.com/istio/istio/releases/download/1.4.0/istio-1.4.0-linux.tar.gz

- **Extract binaries**\n 

 tar -xzf istio-1.4.0-linux.tar.gz

- **You have to instal istioctl command on local before running next step**

 cp bin/istioctl /usr/local/bin/

- **Run bellow command to install default istio by istioctl command** 

istioctl manifest apply -   run this to install istio

- **If you want to generate istio manifest file in case to review it or modify, run below command**

istioctl manifest generate > $HOME/generated-manifest.yaml

-**Once this installed in case your pod to be able to take effect for sidecar container you need to label the namespaces related to that po**

 kubectl label namespace <namespace-name> istio-injection=enabled

- **Then you can deploy your aplication to the labeled namespace, note: you will need to deploy gateway and virtual-service to the same namespace in case your aplication to be accesible externally (find test-example-gtway-vrtsvc-httpbin folder in this repo)**

- **Deploy the test aplication**\n

kubectl create ns test
kubectl label namespace oms istio-injection=enabled
kubectl apply -f samples/httpbin/httpbin.yaml -n test
kubectl apply -f istio-1.4.0/test/gateway-virtual-svc.yml -n test
curl -I  http://$INGRESS_HOST:$INGRESS_PORT/headers
- **Set the ingress ports:**

export INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')

export SECURE_INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].nodePort}')

- **Export the worker node IP**

export INGRESS_HOST='worker-node-IP'

- **To test the aplication try to curl it**

curl -I  http://$INGRESS_HOST:$INGRESS_PORT/headers

- **Also you can check port and host which you created as a variable**

echo ${INGRESS_HOST}

echo ${INGRESS_PORT}

- **To access your aplication externally modify and paste bellow link into your browser**

http://INGRESS_HOST:INGRESS_PORT/headers


# To enable MUTUAL-TLS on your k8s cluster follow bellow instructions

- **Check if there no policies, meshpolicies and destinationrules**

kubectl get policies.authentication.istio.io --all-namespaces

kubectl get meshpolicies.authentication.istio.io

kubectl get destinationrules.networking.istio.io --all-namespaces -o yaml | grep "host:"

- **To set a mesh-wide authentication policy that enables mutual TLS, submit mesh authentication policy (manifests located uder folder mutual-tls)**

kubectl apply -f mustual-tls/global-meshpolicy.yml

- **Create destination rule  from lochal to the istio-system namespace**

kubectl apply -f mustual-tls/global-destinaitonrule.yml

- **Create API destionation rule for local cluster**

kubectl apply -f mustual-tls/api-destinaitonrule.yml

# After you did above steps you can test and confirm if the aplication accesible over browser or curl it, also you need to check if MUTUAL-TLS is enabled using istioctl tool

-**Describe pod using istioctl**

istioctl x describe pod 'POD-NAME' -n 'NAMESPACE-NAME'

- **Check authn tls**

istioctl authn tls-check  'POD-NAME' -n 'NAMESPACE-NAME' 'POD-NAME'.'NAMESPACE-NAME'.svc.cluster.local

- **Check all tls auth**

istioctl authn tls-check  'POD-NAME' -n 'NAMESPACE-NAME'

