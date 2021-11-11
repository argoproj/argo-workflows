package impersonate

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	auth "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	AccessReview(ctx context.Context, namespace string, verb string, resourceGroup string, resourceKind string, resourceName string, subresource string) error
}

type client struct {
	kubeClient kubernetes.Interface
	username   string
}

func NewClient(kubeClient kubernetes.Interface, username string) (Client, error) {
	return &client{
		kubeClient: kubeClient,
		username:   username,
	}, nil
}

func (c *client) AccessReview(ctx context.Context, namespace string, verb string, resourceGroup string, resourceKind string, resourceName string, subresource string) error {
	log.WithFields(log.Fields{
		"Namespace":   namespace,
		"Verb":        verb,
		"Group":       resourceGroup,
		"Resource":    resourceKind,
		"Name":        resourceName,
		"Subresource": subresource,
	}).Debug(fmt.Printf("SubjectAccessReview - %s", c.username))

	review, err := c.kubeClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, &auth.SubjectAccessReview{
		Spec: auth.SubjectAccessReviewSpec{
			User: c.username,
			ResourceAttributes: &auth.ResourceAttributes{
				Namespace:   namespace,
				Verb:        verb,
				Group:       resourceGroup,
				Resource:    resourceKind,
				Name:        resourceName,
				Subresource: subresource,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if !review.Status.Allowed {
		// construct a human-friendly string to represent the resource
		resourceString := ""
		if resourceGroup != "" {
			resourceString += resourceGroup
		}
		if resourceKind != "" {
			resourceString += "/" + resourceKind
		}
		if resourceName != "" {
			resourceString += "/" + resourceName
		}
		if subresource != "" {
			resourceString += "/" + subresource
		}
		resourceString = strings.TrimPrefix(resourceString, "/")

		return status.Error(
			codes.PermissionDenied,
			fmt.Sprintf("user '%s' is not allowed to '%s' %s in namespace '%s'",
				c.username,
				verb,
				resourceString,
				namespace,
			),
		)
	}

	return nil
}
