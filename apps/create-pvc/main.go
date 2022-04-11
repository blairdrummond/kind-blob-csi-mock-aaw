package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

const profileLabel = "profile"
const automountLabel = "blob.aaw.statcan.gc.ca/automount"
const classificationLabel = "data.statcan.gc.ca/classification"

const azureNamespace = "azure-blob-csi-system"

var capacity resource.Quantity = *resource.NewScaledQuantity(100, resource.Giga)

type Instance struct {
	Name           string
	Classification string
	Secret         string
	Capacity       string
	ReadOnly       bool
}

var defaultInstances = `
	{"name": "standard", "classification": "unclassified", "secret": "azure-secret/azure-blob-csi-system", "capacity": "100Gi", "readOnly": false}
	{"name": "premium", "classification": "unclassified", "secret": "azure-secret-premium/azure-blob-csi-system", "capacity": "100Gi", "readOnly": false}
	{"name": "standard-ro", "classification": "protected-b", "secret": "azure-secret/azure-blob-csi-system", "capacity": "100Gi", "readOnly": true}
	{"name": "premium-ro", "classification": "protected-b", "secret": "azure-secret-premium/azure-blob-csi-system", "capacity": "100Gi", "readOnly": true}
`

// name/namespace -> (name, namespace)
func parseSecret(name string) (string, string) {
	// <name>/<namespace> or <name> => <name>/default
	secretName := name
	secretNamespace := azureNamespace
	if strings.Contains(name, "/") {
		split := strings.Split(name, "/")
		secretName = split[0]
		secretNamespace = split[1]
	}
	return secretName, secretNamespace
}

// PV names must be globally unique (they're a cluster resource)
func pvVolumeName(namespace string, instance Instance) string {
	return namespace + "-" + instance.Name
}

// Generate the desired PV Spec
func pvForProfile(namespace string, instance Instance) *corev1.PersistentVolume {

	volumeName := pvVolumeName(namespace, instance)
	secretName, secretNamespace := parseSecret(instance.Secret)

	var accessMode corev1.PersistentVolumeAccessMode
	mountOptions := []string{
		// https://github.com/Azure/azure-storage-fuse/issues/496#issuecomment-704406829
		"-o allow_other",
	}
	if instance.ReadOnly {
		accessMode = corev1.ReadOnlyMany
		// Doesn't work.
		mountOptions = append(mountOptions, "-o ro")
	} else {
		accessMode = corev1.ReadWriteMany
	}

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
			Labels: map[string]string{
				classificationLabel: instance.Classification,
				profileLabel:        namespace,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: capacity,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver: "blob.csi.azure.com",
					NodeStageSecretRef: &corev1.SecretReference{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					ReadOnly: instance.ReadOnly,
					VolumeAttributes: map[string]string{
						"containerName": namespace,
					},
					VolumeHandle: volumeName,
				},
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			MountOptions:                  mountOptions,
			// https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reserving-a-persistentvolume
			ClaimRef: &corev1.ObjectReference{
				Name:      instance.Name,
				Namespace: namespace,
			},
		},
	}

	return pv
}

// Generate the desired PVC Spec
func pvcForProfile(namespace string, instance Instance) *corev1.PersistentVolumeClaim {

	volumeName := pvVolumeName(namespace, instance)
	storageClass := ""

	var accessMode corev1.PersistentVolumeAccessMode
	if instance.ReadOnly {
		accessMode = corev1.ReadOnlyMany
	} else {
		accessMode = corev1.ReadWriteMany
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: namespace,
			Labels: map[string]string{
				classificationLabel: instance.Classification,
				automountLabel:      "true",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: capacity,
				},
			},
			StorageClassName: &storageClass,
			VolumeName:       volumeName,
		},
	}

	return pvc
}

func createPV(client *kubernetes.Clientset, pv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	return client.CoreV1().PersistentVolumes().Create(
		context.Background(),
		pv,
		metav1.CreateOptions{},
	)
}

func createPVC(client *kubernetes.Clientset, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	return client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(
		context.Background(),
		pvc,
		metav1.CreateOptions{},
	)
}

// Sets the global instances variable
func configInstances() []Instance {
	var instances []Instance
	dec := json.NewDecoder(strings.NewReader(defaultInstances))
	for {
		var instance Instance
		err := dec.Decode(&instance)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		fmt.Println(instance)
		instances = append(instances, instance)
	}

	return instances
}

func getBlobClient(client *kubernetes.Clientset, instance Instance) (azblob.ServiceClient, error) {

	fmt.Println(instance.Secret)
	secretName, secretNamespace := parseSecret(instance.Secret)
	secret, err := client.CoreV1().Secrets(secretNamespace).Get(
		context.Background(),
		secretName,
		metav1.GetOptions{},
	)
	if err != nil {
		return azblob.ServiceClient{}, err
	}

	storageAccountName := string(secret.Data["azurestorageaccountname"])
	storageAccountKey := string(secret.Data["azurestorageaccountkey"])

	cred, err := azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		return azblob.ServiceClient{}, err
	}
	service, err := azblob.NewServiceClientWithSharedKey(
		fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName),
		cred,
		nil,
	)
	return service, nil
}

func createContainer(service azblob.ServiceClient, containerName string) error {
	container := service.NewContainerClient(containerName)
	_, err := container.Create(context.Background(), nil)
	return err
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	instances := configInstances()

	for {
		blobClients := map[string]azblob.ServiceClient{}

		// First, branch off of the service client and create a container client.
		for _, instance := range instances {
			if !instance.ReadOnly {
				client, err := getBlobClient(clientset, instance)
				if err != nil {
					panic(err.Error())
				}
				blobClients[instance.Name] = client
			}
		}

		// TODO: Replace with with actual profile logic
		profiles := []string{"alice", "bob"}
		volumes, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		for _, profile := range profiles {
			fmt.Printf("Updating Profile %s...\n", profile)
			claims, err := clientset.CoreV1().PersistentVolumeClaims(profile).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				// Try again in a few seconds.
				fmt.Printf("Failed to list PVCs: %s\n", err.Error())
				break
			}

			for _, instance := range instances {
				if !instance.ReadOnly {
					fmt.Printf("Creating Container %s/%s... ", instance.Name, profile)
					err := createContainer(blobClients[instance.Name], profile)
					if err == nil {
						fmt.Println("Succeeded.")
					} else if strings.Contains(err.Error(), "ContainerAlreadyExists") {
						fmt.Println("Already Exists.")
					} else {
						fmt.Println(err.Error())
					}
				}

				// Create a new PV?
				pv := pvForProfile(profile, instance)
				// Check if it already exists
				pvExists := false
				for _, existingPV := range volumes.Items {
					if pv.Name == existingPV.Name {
						// TODO: Replace with diff/update logic
						pvExists = true
						break
					}
				}
				if !pvExists {
					fmt.Printf("Creating pv %s... ", pv.Name)
					_, err := createPV(clientset, pv)
					if err == nil {
						fmt.Println("Succeeded.")
					} else {
						fmt.Printf("Failed. %s", err.Error())
					}
				}

				// Create new PVC?
				pvc := pvcForProfile(profile, instance)
				// Check if it already exists
				pvcExists := false
				for _, existingPVC := range claims.Items {
					if pvc.Name == existingPVC.Name {
						// TODO: Replace with diff/update logic
						pvcExists = true
						break
					}
				}
				if !pvcExists {
					fmt.Printf("Creating pvc %s/%s... ", pvc.Name, pvc.Namespace)
					_, err := createPVC(clientset, pvc)
					if err == nil {
						fmt.Println("Succeeded.")
					} else {
						fmt.Printf("Failed. %s", err.Error())
					}
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}
