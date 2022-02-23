package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// setupServiceAccount sets up service account and token.
func (woc *wfOperationCtx) setupServiceAccount(pod *apiv1.Pod, tmpl *wfv1.Template) error {
	mainServiceAccountName := woc.getMainServiceAccountName(tmpl)
	automountMainServiceAccountToken := woc.getAutomountMainServiceAccountToken(tmpl)
	executorServiceAccountName := woc.getExecutorServiceAccountName(tmpl)

	log.
		WithField("mainServiceAccountName", mainServiceAccountName).
		WithField("executorServiceAccountName", executorServiceAccountName).
		WithField("pod", pod.Name).
		WithField("template", tmpl.Name).
		Info("set-up service account name")

	secretName, err := woc.getServiceAccountSecretName(mainServiceAccountName)
	if err != nil {
		return err
	}
	pod.Spec.ServiceAccountName = mainServiceAccountName
	pod.Spec.Volumes = append(pod.Spec.Volumes, woc.getMainServiceAccountTokenVolume(secretName))

	if automountMainServiceAccountToken {
		for i, ctr := range pod.Spec.Containers {
			if tmpl.IsMainContainerName(ctr.Name) {
				ctr.VolumeMounts = append(ctr.VolumeMounts, woc.getMainServiceAccountTokenVolumeMount())
				pod.Spec.Containers[i] = ctr
			}
		}
	}

	// executor service account
	{
		secretName, err := woc.getServiceAccountSecretName(executorServiceAccountName)
		if err != nil {
			return err
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, woc.getExecutorServiceAccountTokenVolume(secretName))
	}

	return nil
}
func (woc *wfOperationCtx) getServiceAccountSecretName(saName string) (string, error) {
	key := woc.wf.Namespace + "/" + saName
	store := woc.controller.serviceAccountInformer.GetStore()
	obj, exists, err := store.GetByKey(key)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("failed to find service account %q", key)
	}
	serviceAccount, ok := obj.(*apiv1.ServiceAccount)
	if !ok {
		return "", fmt.Errorf("%T is not *apiv1.ServiceAccount", obj)
	}
	secrets := serviceAccount.Secrets
	if len(secrets) < 1 {
		return "", fmt.Errorf("no secrets in service account %q", key)
	}
	return secrets[0].Name, nil
}

func (woc *wfOperationCtx) getServiceAccountTokenVolume(volumeName, secretName string) apiv1.Volume {
	return apiv1.Volume{
		Name: volumeName,
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}
