CLUSTER := blob-csi-driver
KUBECTL := kubectl --context kind-$(CLUSTER)

AZURE_BLOB_CSI_NAMESPACE := azure-blob-csi-system
K8S_VERSION := "kindest/node:v1.19.11"

install: kind-create azure-secret manifests
destroy: azure-destroy kind-destroy

manifests: manifests/blob-csi-driver.yaml
	kustomize build manifests/ | $(KUBECTL) apply -f -

kind-create:
	kind get clusters | grep -q $(CLUSTER) || \
		kind create cluster --name $(CLUSTER) --image $(K8S_VERSION)

	kind get clusters | grep -q $(CLUSTER) || \
	CID=$$(docker ps | grep $(CLUSTER)-control-plane | awk '{print $$1}') \
	docker exec $(CID) sh -c 'apt-get install libcurl3-gnutls'


# Blob CSI
helm-setup:
	helm repo add blob-csi-driver \
		https://raw.githubusercontent.com/kubernetes-sigs/blob-csi-driver/master/charts || true

manifests/blob-csi-driver.yaml: helm-setup
	mkdir -p $$(dirname $@)
	helm template blob-csi-driver \
		blob-csi-driver/blob-csi-driver \
		--set node.enableBlobfuseProxy=true \
		--namespace kube-system \
		--version v1.9.0 > $@

terraform/terraform.tfstate:
	cd terraform; \
	terraform init; \
	terraform apply

AZURE_DEPS := $(AZURE_BLOB_CSI_NAMESPACE) terraform/terraform.tfstate
$(AZURE_BLOB_CSI_NAMESPACE):
	$(KUBECTL) create ns $@ || true

azure-secret: azure-secret-standard azure-secret-premium azure-secret-fdi-unclassified azure-secret-fdi-prob
azure-secret-standard: $(AZURE_DEPS)
	export NAME=$$(terraform -chdir=terraform output name | tr -d '"'); \
	export KEY=$$(terraform -chdir=terraform output access_key | tr -d '"'); \
	$(KUBECTL) get secret azure-secret -n $(AZURE_BLOB_CSI_NAMESPACE) || \
	$(KUBECTL) create secret generic azure-secret -n $(AZURE_BLOB_CSI_NAMESPACE) \
		--from-literal azurestorageaccountname=$$NAME \
		--from-literal azurestorageaccountkey=$$KEY

azure-secret-premium: $(AZURE_DEPS)
	export NAME=$$(terraform -chdir=terraform output premium_name | tr -d '"'); \
	export KEY=$$(terraform -chdir=terraform output premium_access_key | tr -d '"'); \
	$(KUBECTL) get secret azure-secret-premium -n $(AZURE_BLOB_CSI_NAMESPACE) || \
	$(KUBECTL) create secret generic azure-secret-premium -n $(AZURE_BLOB_CSI_NAMESPACE) \
		--from-literal azurestorageaccountname=$$NAME \
		--from-literal azurestorageaccountkey=$$KEY

azure-secret-fdi-prob: $(AZURE_DEPS)
	export NAME=$$(terraform -chdir=terraform output fdi_prob_name | tr -d '"'); \
	export KEY=$$(terraform -chdir=terraform output fdi_prob_access_key | tr -d '"'); \
	$(KUBECTL) get secret azure-secret-fdi-prob -n $(AZURE_BLOB_CSI_NAMESPACE) || \
	$(KUBECTL) create secret generic azure-secret-fdi-prob -n $(AZURE_BLOB_CSI_NAMESPACE) \
		--from-literal azurestorageaccountname=$$NAME \
		--from-literal azurestorageaccountkey=$$KEY

azure-secret-fdi-unclassified: $(AZURE_DEPS)
	export NAME=$$(terraform -chdir=terraform output fdi_unclassified_name | tr -d '"'); \
	export KEY=$$(terraform -chdir=terraform output fdi_unclassified_access_key | tr -d '"'); \
	$(KUBECTL) get secret azure-secret-fdi-unclassified -n $(AZURE_BLOB_CSI_NAMESPACE) || \
	$(KUBECTL) create secret generic azure-secret-fdi-unclassified -n $(AZURE_BLOB_CSI_NAMESPACE) \
		--from-literal azurestorageaccountname=$$NAME \
		--from-literal azurestorageaccountkey=$$KEY



# Destroy
azure-destroy:
	cd terraform; \
	terraform destroy

kind-delete:
	kind delete cluster --name $(CLUSTER)
