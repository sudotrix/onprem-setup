FROM alpine:3.10
##RKE Provider V1.0.0-rc3 is used ##

ENV TERRAFORM_VERSION=0.13.7
ENV KUBECTL_VERSION=v1.18.2
ENV GOSS_VERSION=v0.3.11
ENV RKE_PROVIDER_VERSION=1.2.0
ENV ISTIO_VERSION=1.9.2
ENV YTT_VERSION=v0.27.1
ENV JQ=v1.6-r0
ENV DIG=9.14.12
ENV PYTHON=2.7.17
ENV PATH $PATH:/app/google-cloud-sdk/bin
ENV HELM_VERSION=v3.2.4
ENV RANCHER_CLI_VERSION=v2.4.11
ENV GOOGLE_PROVIDER=3.65.0
ENV LOCAL_PROVIDER=1.4.0
ENV NULL_PROVIDER=2.1.2
ENV RANCHER_PROVIDER=1.13.0
ENV RANCHER_CLI_VERSION=v2.4.11

WORKDIR /app

RUN apk --no-cache update \
  && apk upgrade \
  && apk --no-cache add curl git wget unzip openssh jq bind-tools python bash \
  && curl -O -A "Mozilla Chrome Safari" https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-294.0.0-linux-x86.tar.gz \
  && tar -xzvf google-cloud-sdk-294.0.0-linux-x86.tar.gz \
  && rm google-cloud-sdk-294.0.0-linux-x86.tar.gz \
  && wget /app https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip -O terraform.zip ; unzip /app/terraform.zip \
  && chmod +x /app/terraform; mv /app/terraform /usr/local/bin/; rm /app/terraform.zip \
  && wget --no-check-certificate -P /app  https://github.com/rancher/terraform-provider-rke/releases/download/v"$RKE_PROVIDER_VERSION"/terraform-provider-rke_"$RKE_PROVIDER_VERSION"_linux_amd64.zip ; unzip /app/terraform-provider-rke_"$RKE_PROVIDER_VERSION"_linux_amd64.zip \
  && mkdir -p /root/.kube/ /root/.terraform.d/plugins; mv /app/terraform-provider-rke_v"$RKE_PROVIDER_VERSION" /root/.terraform.d/plugins; rm /app/terraform-provider-rke_"$RKE_PROVIDER_VERSION"_linux_amd64.zip \
  && curl -LO https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl; chmod +x /app/kubectl; mv /app/kubectl /usr/local/bin/ \
  && curl -L https://github.com/aelsabbahy/goss/releases/download/$GOSS_VERSION/goss-linux-amd64 -o /usr/local/bin/goss; chmod +x /usr/local/bin/goss \
  && wget https://github.com/istio/istio/releases/download/$ISTIO_VERSION/istioctl-$ISTIO_VERSION-linux-amd64.tar.gz; tar -xzf istioctl-$ISTIO_VERSION-linux-amd64.tar.gz \
  && mv istioctl /usr/local/bin/; chmod +x /usr/local/bin/istioctl \
  && rm istioctl-$ISTIO_VERSION-linux-amd64.tar.gz \
  && curl -L https://github.com/k14s/ytt/releases/download/$YTT_VERSION/ytt-linux-amd64 -o /usr/local/bin/ytt; chmod +x /usr/local/bin/ytt \
  && wget https://releases.hashicorp.com/vault/1.3.0/vault_1.3.0_linux_amd64.zip \
  && unzip vault_1.3.0_linux_amd64.zip; mv ./vault /usr/local/bin; rm vault_1.3.0_linux_amd64.zip; chmod +x /usr/local/bin/vault \
  && wget --no-check-certificate -P /app https://get.helm.sh/helm-$HELM_VERSION-linux-386.tar.gz ; tar -xzvf /app/helm-$HELM_VERSION-linux-386.tar.gz \
  && mv /app/linux-386/helm /usr/local/bin/helm; rm /app/helm-$HELM_VERSION-linux-386.tar.gz \
  && helm plugin install https://github.com/viglesiasce/helm-gcs.git --version v0.2.0 \
  && curl -sSL "https://github.com/rancher/cli/releases/download/${RANCHER_CLI_VERSION}/rancher-linux-amd64-${RANCHER_CLI_VERSION}.tar.gz" | tar -xz -C /usr/local/bin/ --strip-components=2 \
  && apk add --no-cache python3 py3-pip \
  && wget https://releases.hashicorp.com/terraform-provider-google/"$GOOGLE_PROVIDER"/terraform-provider-google_"$GOOGLE_PROVIDER"_linux_amd64.zip; unzip terraform-provider-google_"$GOOGLE_PROVIDER"_linux_amd64.zip \
  && mv /app/terraform-provider-google_v"$GOOGLE_PROVIDER"_x5 /root/.terraform.d/plugins; rm -rf /app/terraform-provider-google_"$GOOGLE_PROVIDER"_linux_amd64.zip \
  && wget https://releases.hashicorp.com/terraform-provider-local/"$LOCAL_PROVIDER"/terraform-provider-local_"$LOCAL_PROVIDER"_linux_amd64.zip; unzip terraform-provider-local_"$LOCAL_PROVIDER"_linux_amd64.zip \
  && mv /app/terraform-provider-local_v"$LOCAL_PROVIDER"_x4 /root/.terraform.d/plugins; rm -rf /app/terraform-provider-local_"$LOCAL_PROVIDER"_linux_amd64.zip \
  && wget https://releases.hashicorp.com/terraform-provider-null/"$NULL_PROVIDER"/terraform-provider-null_"$NULL_PROVIDER"_linux_amd64.zip; unzip terraform-provider-null_"$NULL_PROVIDER"_linux_amd64.zip \
  && mv /app/terraform-provider-null_v"$NULL_PROVIDER"_x4 /root/.terraform.d/plugins; rm -rf /app/terraform-provider-null_"$NULL_PROVIDER"_linux_amd64.zip \
  && wget https://releases.hashicorp.com/terraform-provider-rancher2/"$RANCHER_PROVIDER"/terraform-provider-rancher2_"$RANCHER_PROVIDER"_linux_amd64.zip; unzip terraform-provider-rancher2_"$RANCHER_PROVIDER"_linux_amd64.zip \
  && mv /app/terraform-provider-rancher2_v"$RANCHER_PROVIDER" /root/.terraform.d/plugins; rm -rf terraform-provider-rancher2_"$RANCHER_PROVIDER"_linux_amd64.zip \
  && curl -sSL "https://github.com/rancher/cli/releases/download/${RANCHER_CLI_VERSION}/rancher-linux-amd64-${RANCHER_CLI_VERSION}.tar.gz" | tar -xz -C /usr/local/bin/ --strip-components=2 \
  && apk add --no-cache python3 py3-pip


# echo "copying scripts" ##
COPY goss.yaml goss_test.sh ./

RUN chmod +x goss_test.sh

