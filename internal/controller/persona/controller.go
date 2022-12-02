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

package persona

import (
	"context"
	"time"

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
	service "github.com/VariableExp0rt/powerbroker/internal/service"
	personasvc "github.com/VariableExp0rt/powerbroker/internal/service/persona"
	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
)

const (
	errNotPersona   = "managed resource is not a Persona custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

var (
	errNewService = errors.New("cannot create new service client")
)

// Setup adds a controller that reconciles Persona managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.PersonaGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Persona{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PersonaGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				util:  &connectorHelper{},
			}),
			managed.WithCreationGracePeriod(10*time.Second),
			managed.WithInitializers(managed.NewDefaultProviderConfig(mgr.GetClient())),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

type Connector interface {
	GetService(repo service.Repository) personasvc.Service
	ExtractCredentials(context.Context, v1.CredentialsSource, client.Client, v1.CommonCredentialSelectors) ([]byte, error)
}

type connectorHelper struct{}

func (c *connectorHelper) GetService(repo service.Repository) personasvc.Service {
	return personasvc.NewService(repo)
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
	cr, ok := mg.(*v1alpha1.Persona)
	if !ok {
		return nil, errors.New(errNotPersona)
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

	var service personasvc.Service
	switch pc.Spec.Storage.Type {
	case "neo4j":
		store, err := storage.NewNeo4jStorage(data)
		if err != nil {
			return nil, errors.Wrap(err, "client")
		}
		service = personasvc.NewService(store)
	default:
		return nil, errNewService
	}

	return &external{service: service, kube: c.kube}, nil
}

type external struct {
	kube    client.Client
	service personasvc.Service
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Persona)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPersona)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	currentRefs := cr.Spec.ForProvider.DeepCopy().PermissionSets

	resp, err := e.service.GetPersona(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot get persona")
	}

	cr.Status.AtProvider = generatePersonaObservation(resp)
	switch cr.Status.AtProvider.Status {
	case string(storetypes.StatusAvailable):
		cr.SetConditions(v1.Available())
	case string(storetypes.StatusUnavailable):
		cr.SetConditions(v1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: cmp.Equal(currentRefs, resp.References),
		Diff:             cmp.Diff(currentRefs, resp.References),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Persona)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPersona)
	}

	references := cr.Spec.ForProvider.DeepCopy().PermissionSets

	cr.SetConditions(v1.Creating())
	uuid, err := e.service.CreatePersona(
		cr.Spec.ForProvider.Name,
		references,
	)

	return postCreate(cr, managed.ExternalCreation{ExternalNameAssigned: true}, uuid, err)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Persona)
	if !ok {
		return managed.ExternalUpdate{}, nil
	}

	references := cr.Spec.ForProvider.DeepCopy().PermissionSets

	err := e.service.UpdatePersona(
		cr.GetName(),
		meta.GetExternalName(cr),
		references,
	)

	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update persona")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Persona)
	if !ok {
		return nil
	}

	cr.SetConditions(v1.Deleting())
	err := e.service.DeletePersona(meta.GetExternalName(cr))

	return errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot delete persona")
}

func generatePersonaObservation(r *svctypes.GetPersonaResponse) v1alpha1.PersonaObservation {
	return v1alpha1.PersonaObservation{
		NodeID: r.NodeID,
		Status: string(r.Status),
	}
}

func postCreate(cr *v1alpha1.Persona, ec managed.ExternalCreation, uuid string, err error) (managed.ExternalCreation, error) {
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create persona")
	}

	meta.SetExternalName(cr, uuid)
	return ec, nil
}
