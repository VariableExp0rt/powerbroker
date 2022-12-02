/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package permissionset

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	apisv1alpha1 "github.com/VariableExp0rt/powerbroker/apis/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/controller/features"
	svc "github.com/VariableExp0rt/powerbroker/internal/service"
	permissionsetsvc "github.com/VariableExp0rt/powerbroker/internal/service/permissionset"
	pwrbrkrtypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	storage "github.com/VariableExp0rt/powerbroker/internal/storage"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
)

const (
	errNotPermissionSet = "managed resource is not a PermissionSet custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
)

var (
	errNewService = errors.New("cannot create new service client")
)

var _ Connector = &connectorHelper{}

// Setup adds a controller that reconciles PermissionSet managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.PermissionSetGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.PermissionSet{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PermissionSetGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				util:  &connectorHelper{},
			}),
			managed.WithCreationGracePeriod(10*time.Second),
			managed.WithInitializers(managed.NewDefaultProviderConfig(mgr.GetClient())),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithPollInterval(o.PollInterval),
			managed.WithConnectionPublishers(cps...)))
}

type Connector interface {
	GetService(svc.Repository) permissionsetsvc.Service
	ExtractCredentials(context.Context, v1.CredentialsSource, client.Client, v1.CommonCredentialSelectors) ([]byte, error)
}

type connectorHelper struct{}

func (c *connectorHelper) GetService(repo svc.Repository) permissionsetsvc.Service {
	return permissionsetsvc.NewService(repo)
}

func (c *connectorHelper) ExtractCredentials(ctx context.Context, source v1.CredentialsSource, kube client.Client, selector v1.CommonCredentialSelectors) ([]byte, error) {
	return resource.CommonCredentialExtractor(ctx, source, kube, selector)
}

type connector struct {
	kube  client.Client
	usage resource.Tracker
	util  Connector
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.PermissionSet)
	if !ok {
		return nil, errors.New(errNotPermissionSet)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.New(errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	//TODO(liambaker): this will not extract stringData from configMap
	cd := pc.Spec.Credentials
	data, err := c.util.ExtractCredentials(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	var service permissionsetsvc.Service
	switch pc.Spec.Storage.Type {
	case "neo4j":
		store, err := storage.NewNeo4jStorage(data)
		if err != nil {
			return nil, errors.Wrap(err, "client")
		}
		service = permissionsetsvc.NewService(store)
	default:
		return nil, errNewService
	}

	return &external{service: service, kube: c.kube}, nil
}

type external struct {
	kube    client.Client
	service permissionsetsvc.Service
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.PermissionSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPermissionSet)
	}

	ext := meta.GetExternalName(cr)
	if ext == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	desiredBinding := cr.Spec.ForProvider.BindTo.DeepCopy()

	// TODO: merge the observed binding into this "api's" response
	// as we're sort of manufacturing a status here
	resp, err := e.service.GetPermissionSet(ext)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(
			resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err),
			"cannot get permissionset")
	}

	cr.Status.AtProvider = generatePermissionSetObservation(resp)

	return postObserve(cr, managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: cmp.Equal(*desiredBinding, resp.Binding,
			cmpopts.IgnoreTypes([]v1.Reference{}, []v1.Selector{}),
		),
		Diff: cmp.Diff(*desiredBinding, resp.Binding,
			cmpopts.IgnoreTypes([]v1.Reference{}, []v1.Selector{}),
		),
	}, err)
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.PermissionSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPermissionSet)
	}

	cr.SetConditions(v1.Creating())
	uuid, err := e.service.CreatePermissionSet(
		cr.Name,
		cr.Spec.ForProvider.BindTo,
	)

	return postCreate(cr, managed.ExternalCreation{ExternalNameAssigned: true}, uuid, err)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.PermissionSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPermissionSet)
	}

	// TODO(liambaker): fix the fact that it does not update alias, class
	// just pass the entire BindTo here and handle alias and class too
	err := e.service.UpdatePermissionSet(
		meta.GetExternalName(cr),
		cr.GetName(),
		cr.Spec.ForProvider.BindTo,
	)

	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update permissionset")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.PermissionSet)
	if !ok {
		return errors.New(errNotPermissionSet)
	}

	cr.SetConditions(v1.Deleting())
	err := e.service.DeletePermissionSet(meta.GetExternalName(cr))

	return errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot delete permissionset")
}

func generatePermissionSetObservation(r *pwrbrkrtypes.GetPermissionSetResponse) v1alpha1.PermissionSetObservation {
	return v1alpha1.PermissionSetObservation{
		NodeID: r.NodeID,
		Status: string(r.Status),
	}
}

func postObserve(cr *v1alpha1.PermissionSet, obs managed.ExternalObservation, err error) (managed.ExternalObservation, error) {
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.SetConditions(v1.Available())
	return obs, nil
}

func postCreate(cr *v1alpha1.PermissionSet, ec managed.ExternalCreation, uuid string, err error) (managed.ExternalCreation, error) {
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create permissionset")
	}

	meta.SetExternalName(cr, uuid)
	return ec, nil
}
