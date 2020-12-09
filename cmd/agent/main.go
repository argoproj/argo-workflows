package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var (
		clientConfig clientcmd.ClientConfig
		secure       bool
		logLevel     string
	)
	listenAndServe := func(handlerFunc http.Handler) error {
		// 24368 = "agent" on an old phone keypad
		addr := ":24368"
		log.Infof("starting to listen on %v", addr)
		if secure {
			return http.ListenAndServeTLS(addr, "agent.crt", "agent.key", handlerFunc)
		} else {
			log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://argoproj.github.io/argo/agent/")
			return http.ListenAndServe(addr, handlerFunc)
		}
	}
	cmd := cobra.Command{
		PreRun: func(cmd *cobra.Command, args []string) {
			cli.SetLogLevel(logLevel)
		},
		Run: func(cmd *cobra.Command, args []string) {
			r, err := clientConfig.ClientConfig()
			if err != nil {
				log.Fatal(err)
			}
			clientset, err := kubernetes.NewForConfig(r)
			if err != nil {
				log.Fatal(err)
			}
			secret, err := clientset.CoreV1().Secrets(os.Getenv("NAMESPACE")).Get("agent", metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}
			requiredToken := secret.Data["token"]
			getHealth := func(w http.ResponseWriter, r *http.Request) { send(w, http.StatusOK, nil) }
			listPods := func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				q := r.URL.Query()
				listOptions := metav1.ListOptions{
					LabelSelector:   q.Get("labelSelector"),
					ResourceVersion: q.Get("resourceVersion"),
					Watch:           q.Get("watch") == "true",
				}
				if !(q.Get("watch") == "true") {
					podList, err := clientset.CoreV1().Pods(vars["namespace"]).List(listOptions)
					if err != nil {
						sendErr(w, err)
					} else {
						send(w, http.StatusOK, podList)
					}
				} else {
					podList, err := clientset.CoreV1().Pods(vars["namespace"]).Watch(listOptions)
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
			createPod := func(w http.ResponseWriter, r *http.Request) {
				pod := &corev1.Pod{}
				err := json.NewDecoder(r.Body).Decode(pod)
				if err != nil {
					sendErr(w, err)
					return
				}
				vars := mux.Vars(r)
				pod, err = clientset.CoreV1().Pods(vars["namespace"]).Create(pod)
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusCreated, pod)
				}
			}
			deletePods := func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				q := r.URL.Query()
				listOptions := metav1.ListOptions{LabelSelector: q.Get("labelSelector")}
				err = clientset.CoreV1().Pods(vars["namespace"]).DeleteCollection(&metav1.DeleteOptions{}, listOptions)
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusCreated, nil)
				}
			}
			getPod := func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				pod, err := clientset.CoreV1().Pods(vars["namespace"]).Get(vars["name"], metav1.GetOptions{})
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusOK, pod)
				}
			}
			updatePod := func(w http.ResponseWriter, r *http.Request) {
				pod := &corev1.Pod{}
				err := json.NewDecoder(r.Body).Decode(pod)
				if err != nil {
					sendErr(w, err)
					return
				}
				vars := mux.Vars(r)
				pod, err = clientset.CoreV1().Pods(vars["namespace"]).Update(pod)
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusOK, pod)
				}
			}
			patchPod := func(w http.ResponseWriter, r *http.Request) {
				data, err := ioutil.ReadAll(r.Body)
				if err != nil {
					sendErr(w, err)
					return
				}
				vars := mux.Vars(r)
				contentType := r.Header.Get("Content-Type")
				pod, err := clientset.CoreV1().Pods(vars["namespace"]).Patch(vars["name"], types.PatchType(contentType), data)
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusOK, pod)
				}
			}
			deletePod := func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				err := clientset.CoreV1().Pods(vars["namespace"]).Delete(vars["name"], &metav1.DeleteOptions{})
				if err != nil {
					sendErr(w, err)
				} else {
					send(w, http.StatusOK, nil)
				}
			}
			router := mux.NewRouter().StrictSlash(true)
			router.HandleFunc("/health", getHealth)
			// kubectl get --raw /openapi/v2
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods", listPods).Methods("GET")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods", createPod).Methods("POST")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods", deletePods).Methods("DELETE")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", getPod).Methods("GET")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", updatePod).Methods("PUT")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", patchPod).Methods("PATCH")
			router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", deletePod).Methods("DELETE")

			log.Fatal(listenAndServe(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log.Infof("%s %s", r.Method, r.URL.String())
				authorization := r.Header.Get("Authorization")
				requiredAuthorization := "Bearer " + base64.StdEncoding.EncodeToString(requiredToken)
				if r.URL.Path != "/health" && authorization != requiredAuthorization {
					log.WithFields(log.Fields{"authorization": authorization, "requiredAuthorization": requiredAuthorization}).Debug()
					sendErr(w, errors.NewUnauthorized("wrong authorization token"))
					return
				}
				router.ServeHTTP(w, r)
			})))
		}}
	clientConfig = kubecli.AddKubectlFlagsToCmd(&cmd)
	cmd.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	cmd.PersistentFlags().BoolVarP(&secure, "secure", "e", true, "Serve using TLS")
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func sendErr(w http.ResponseWriter, err error) {
	switch v := err.(type) {
	case *errors.StatusError:
		send(w, int(v.Status().Code), v.Status())
	default:
		send(w, http.StatusInternalServerError, errors.NewInternalError(err))
	}
}

func send(w http.ResponseWriter, code int, v interface{}) {
	if s, ok := v.(metav1.Status); ok {
		log.Warnf("%v: %s", code, s.Message)
	} else {
		log.Info(code)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
