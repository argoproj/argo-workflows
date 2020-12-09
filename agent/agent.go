package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Agent interface {
	Run() error
}

type agent struct {
	kube                  kubernetes.Interface
	secure                bool
	requiredAuthorization string
}

func NewAgent(kube kubernetes.Interface, namespace string, secure bool) (Agent, error) {
	secret, err := kube.CoreV1().Secrets(namespace).Get("agent", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	requiredAuthorization := "Bearer " + base64.StdEncoding.EncodeToString(secret.Data["token"])
	return &agent{kube, secure, requiredAuthorization}, nil
}

func (a *agent) getHealth(w http.ResponseWriter, _ *http.Request) { send(w, http.StatusOK, nil) }

func (a *agent) listPods(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	q := r.URL.Query()
	listOptions := metav1.ListOptions{
		LabelSelector:   q.Get("labelSelector"),
		ResourceVersion: q.Get("resourceVersion"),
		Watch:           q.Get("watch") == "true",
	}
	if !(q.Get("watch") == "true") {
		podList, err := a.kube.CoreV1().Pods(vars["namespace"]).List(listOptions)
		if err != nil {
			sendErr(w, err)
		} else {
			send(w, http.StatusOK, podList)
		}
	} else {
		podList, err := a.kube.CoreV1().Pods(vars["namespace"]).Watch(listOptions)
		if err != nil {
			sendErr(w, err)
			return
		}
		defer podList.Stop()
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		flusher, ok := w.(http.Flusher)
		if !ok {
			sendErr(w, fmt.Errorf("not a flusher"))
		}
		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-podList.ResultChan():
				log.Info(event.Type)
				object := event.Object
				switch v := object.(type) {
				case *corev1.Pod:
					v.APIVersion = "v1"
					v.Kind = "Pod"
				case *metav1.Status:
					v.APIVersion = "v1"
					v.Kind = "Status"
				}
				_ = encoder.Encode(map[string]interface{}{"type": event.Type, "object": object})
				_, _ = w.Write([]byte("\r\n"))
				flusher.Flush()

			}
		}
	}
}
func (a *agent) createPod(w http.ResponseWriter, r *http.Request) {
	pod := &corev1.Pod{}
	err := json.NewDecoder(r.Body).Decode(pod)
	if err != nil {
		sendErr(w, err)
		return
	}
	vars := mux.Vars(r)
	pod, err = a.kube.CoreV1().Pods(vars["namespace"]).Create(pod)
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusCreated, pod)
	}
}
func (a *agent) deletePods(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	q := r.URL.Query()
	listOptions := metav1.ListOptions{LabelSelector: q.Get("labelSelector")}
	err := a.kube.CoreV1().Pods(vars["namespace"]).DeleteCollection(&metav1.DeleteOptions{}, listOptions)
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusCreated, nil)
	}
}
func (a *agent) getPod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pod, err := a.kube.CoreV1().Pods(vars["namespace"]).Get(vars["name"], metav1.GetOptions{})
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusOK, pod)
	}
}
func (a *agent) updatePod(w http.ResponseWriter, r *http.Request) {
	pod := &corev1.Pod{}
	err := json.NewDecoder(r.Body).Decode(pod)
	if err != nil {
		sendErr(w, err)
		return
	}
	vars := mux.Vars(r)
	pod, err = a.kube.CoreV1().Pods(vars["namespace"]).Update(pod)
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusOK, pod)
	}
}
func (a *agent) patchPod(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sendErr(w, err)
		return
	}
	vars := mux.Vars(r)
	contentType := r.Header.Get("Content-Type")
	pod, err := a.kube.CoreV1().Pods(vars["namespace"]).Patch(vars["name"], types.PatchType(contentType), data)
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusOK, pod)
	}
}

func (a *agent) deletePod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := a.kube.CoreV1().Pods(vars["namespace"]).Delete(vars["name"], &metav1.DeleteOptions{})
	if err != nil {
		sendErr(w, err)
	} else {
		send(w, http.StatusOK, nil)
	}
}

func (a *agent) authenticationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		if r.URL.Path != "/health" && authorization != a.requiredAuthorization {
			log.WithFields(log.Fields{"authorization": authorization, "requiredAuthorization": a.requiredAuthorization}).Debug()
			sendErr(w, errors.NewUnauthorized("wrong authorization token"))
			return
		}
		next(w, r)
	}
}

func (a *agent) listenAndServe(handlerFunc http.Handler) error {
	// 24368 = "agent" on an old phone keypad
	addr := ":24368"
	log.Infof("starting to listen on %v", addr)
	if a.secure {
		return http.ListenAndServeTLS(addr, "agent.crt", "agent.key", handlerFunc)
	} else {
		log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://argoproj.github.io/argo/agent/")
		return http.ListenAndServe(addr, handlerFunc)
	}
}
func (a *agent) Run() error {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/health", a.getHealth)

	// kubectl get --raw /openapi/v2
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods", a.listPods).Methods("GET")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods", a.deletePods).Methods("DELETE")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods", a.createPod).Methods("POST")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", a.getPod).Methods("GET")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", a.updatePod).Methods("PUT")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", a.patchPod).Methods("PATCH")
	router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", a.deletePod).Methods("DELETE")

	return a.listenAndServe(handlers.RecoveryHandler()(handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, a.authenticationMiddleware(router.ServeHTTP)))))
}
