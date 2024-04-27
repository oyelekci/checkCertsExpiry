package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var namespace = "default"
var containerName = "nginx"
var certPath = "/etc/ssl/certs/tls.crt"

func getPods(clientset *kubernetes.Clientset) (*v1.PodList, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return podList, nil
}

func getCert(pod v1.Pod) (x509.Certificate, error) {
	cmd := exec.Command("kubectl", "exec", "-it", pod.Name, "-c", containerName, "--", "cat", certPath)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return x509.Certificate{}, err
	}

	block, _ := pem.Decode(out.Bytes())
	if block == nil {
		return x509.Certificate{}, errors.New("failed to decode PEM data")
	}

	if block.Type != "CERTIFICATE" {
		return x509.Certificate{}, errors.New("PEM data is not a certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return x509.Certificate{}, err
	}

	return *cert, nil
}

func main() {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.ExpandEnv("$HOME/.kube/config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podList, err := getPods(clientset)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Expiry dates of certs in %s:\n", namespace)
	for _, pod := range podList.Items {
		cert, err := getCert(pod)
		if err != nil {
			fmt.Printf("could not get certificate for pod %s because %s\n", pod.Name, err)
			continue
		}
		fmt.Println("-", pod.Name, "-", cert.NotAfter.Format("2006-01-02T15:04:05Z"))
	}
}
