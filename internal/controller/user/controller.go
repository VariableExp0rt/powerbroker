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

package user

import (
	"context"

	"github.com/google/go-cmp/cmp"
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
	usersvc "github.com/VariableExp0rt/powerbroker/internal/service/user"
	storage "github.com/VariableExp0rt/powerbroker/internal/storage"

	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage/neo4j/transaction"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
)

const (
	errNotUser      = "managed resource is not a User custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

var (
	errNewService = errors.New("cannot create new service client")
)

type Connector interface {
	GetService(repo svc.Repository) usersvc.Service
	ExtractCredentials(context.Context, v1.CredentialsSource, client.Client, v1.CommonCredentialSelectors) ([]byte, error)
}

type connectorHelper struct{}

func (c *connectorHelper) GetService(repo svc.Repository) usersvc.Service {
	return usersvc.NewService(repo)
}

func (c *connectorHelper) ExtractCredentials(ctx context.Context, source v1.CredentialsSource, kube client.Client, selector v1.CommonCredentialSelectors) ([]byte, error) {
	return resource.CommonCredentialExtractor(ctx, source, kube, selector)
}

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.User{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.UserGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				util:  &connectorHelper{}}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

type connector struct {
	kube  client.Client
	usage resource.Tracker
	util  Connector
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.New(errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderReference().Name}, pc); err != nil {
		return nil, errors.New(errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := c.util.ExtractCredentials(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	var service usersvc.Service
	switch pc.Spec.Storage.Type {
	case "neo4j":
		store, err := storage.NewNeo4jStorage(data)
		if err != nil {
			return nil, errors.Wrap(err, "client")
		}
		service = usersvc.NewService(store)
	default:
		return nil, errNewService
	}

	return &external{service: service}, nil
}

type external struct {
	kube    client.Client
	service usersvc.Service
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	ext := meta.GetExternalName(cr)
	if ext == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	refs := cr.Spec.ForProvider.DeepCopy().Personas
	resp, err := e.service.GetUser(
		ext,
	)
	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot get user")
	}

	cr.Status.AtProvider = generateUserObservation(resp)
	switch cr.Status.AtProvider.Status {
	case transaction.StatusAvailable:
		cr.SetConditions(v1.Available())
	case transaction.StatusUnavailable:
		cr.SetConditions(v1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: cmp.Equal(refs, resp.References),
		Diff:             cmp.Diff(refs, resp.References),
	}, err
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	refs := cr.Spec.ForProvider.DeepCopy().Personas

	cr.SetConditions(v1.Creating())
	uuid, err := e.service.CreateUser(
		cr.Spec.ForProvider.Name,
		refs,
	)

	return postCreate(cr, managed.ExternalCreation{ExternalNameAssigned: true}, uuid, err)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	references := cr.Spec.ForProvider.DeepCopy().Personas

	err := e.service.UpdateUser(
		cr.GetName(),
		meta.GetExternalName(cr),
		references,
	)

	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update user")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return errors.New(errNotUser)
	}

	cr.SetConditions(v1.Deleting())
	err := e.service.DeleteUser(meta.GetExternalName(cr))

	return errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot delete user")
}

func generateUserObservation(r *svctypes.GetUserResponse) v1alpha1.UserObservation {
	return v1alpha1.UserObservation{
		NodeID: r.NodeID,
		Status: string(r.Status),
	}
}

func postCreate(cr *v1alpha1.User, ec managed.ExternalCreation, uuid string, err error) (managed.ExternalCreation, error) {
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create user")
	}

	meta.SetExternalName(cr, uuid)
	return ec, nil
}
