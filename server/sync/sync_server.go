package sync

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

type syncServer struct {
}

func NewSyncServer() *syncServer {
	return &syncServer{}
}

func (s *syncServer) CreateSyncLimit(ctx context.Context, req *syncpkg.CreateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	if req.SizeLimit <= 0 {
		return nil, sutils.ToStatusError(fmt.Errorf("size limit must be greater than zero"), codes.InvalidArgument)
	}

	kubeClient := auth.GetKubeClient(ctx)

	configmapGetter := kubeClient.CoreV1().ConfigMaps(req.Namespace)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Data: map[string]string{
			req.Key: fmt.Sprint(req.SizeLimit),
		},
	}

	cm, err := configmapGetter.Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return s.updateSyncLimit(ctx, &syncpkg.UpdateSyncLimitRequest{
				Name:      req.Name,
				Namespace: req.Namespace,
				Key:       req.Key,
				SizeLimit: req.SizeLimit,
			}, false)
		}

		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return &syncpkg.SyncLimitResponse{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Key:       req.Key,
		SizeLimit: req.SizeLimit,
	}, nil
}

func (s *syncServer) GetSyncLimit(ctx context.Context, req *syncpkg.GetSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	kubeClient := auth.GetKubeClient(ctx)

	configmapGetter := kubeClient.CoreV1().ConfigMaps(req.Namespace)

	cm, err := configmapGetter.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	sizeLimit, ok := cm.Data[req.Key]
	if !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("key %s not found in configmap %s/%s", req.Key, cm.Namespace, cm.Name), codes.NotFound)
	}

	parsedSizeLimit, err := strconv.Atoi(sizeLimit)
	if err != nil {
		return nil, sutils.ToStatusError(fmt.Errorf("invalid size limit format for key %s in configmap %s/%s", req.Key, cm.Namespace, cm.Name), codes.InvalidArgument)
	}

	return &syncpkg.SyncLimitResponse{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Key:       req.Key,
		SizeLimit: int32(parsedSizeLimit),
	}, nil
}

func (s *syncServer) UpdateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	if req.SizeLimit <= 0 {
		return nil, sutils.ToStatusError(fmt.Errorf("size limit must be greater than zero"), codes.InvalidArgument)
	}

	return s.updateSyncLimit(ctx, req, true)
}

func (s *syncServer) updateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest, shouldFieldExist bool) (*syncpkg.SyncLimitResponse, error) {
	kubeClient := auth.GetKubeClient(ctx)

	configmapGetter := kubeClient.CoreV1().ConfigMaps(req.Namespace)

	cm, err := configmapGetter.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	if _, ok := cm.Data[req.Key]; shouldFieldExist && !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("key %s not found in configmap %s/%s - please create it first", req.Key, cm.Namespace, cm.Name), codes.NotFound)
	}

	cm.Data[req.Key] = strconv.Itoa(int(req.SizeLimit))

	cm, err = configmapGetter.Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return &syncpkg.SyncLimitResponse{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Key:       req.Key,
		SizeLimit: req.SizeLimit,
	}, nil
}

func (s *syncServer) DeleteSyncLimit(ctx context.Context, req *syncpkg.DeleteSyncLimitRequest) (*syncpkg.DeleteSyncLimitResponse, error) {
	fmt.Printf("Deleting sync limit for ConfigMap %s in namespace %s with key %s\n", req.Name, req.Namespace, req.Key)

	kubeClient := auth.GetKubeClient(ctx)

	configmapGetter := kubeClient.CoreV1().ConfigMaps(req.Namespace)

	cm, err := configmapGetter.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	delete(cm.Data, req.Key)

	_, err = configmapGetter.Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return &syncpkg.DeleteSyncLimitResponse{}, nil
}
