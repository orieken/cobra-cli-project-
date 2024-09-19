package awesome_uatu

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	http.HandleFunc("/status", handleStatus)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func getClientset() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if home := homedir.HomeDir(); home != "" {
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func getPods(clientset *kubernetes.Clientset) ([]PodStatus, error) {
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var statuses []PodStatus
	for _, pod := range pods.Items {
		statuses = append(statuses, PodStatus{
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
		})
	}
	return statuses, nil
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	clientset, err := getClientset()
	if err != nil {
		http.Error(w, "Failed to get Kubernetes clientset: "+err.Error(), http.StatusInternalServerError)
		return
	}
	statuses, err := getPods(clientset)
	if err != nil {
		http.Error(w, "Failed to get pods: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := json.Marshal(statuses)
	if err != nil {
		http.Error(w, "Failed to marshal pod statuses: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

type PodStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
