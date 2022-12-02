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

package team

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
	"github.com/VariableExp0rt/powerbroker/internal/service"
	teamsvc "github.com/VariableExp0rt/powerbroker/internal/service/team"
	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
)

const (
	errNotTeam      = "managed resource is not a Team custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
)

var (
	errNewService = errors.New("cannot create new service client")
)

// Setup adds a controller that reconciles Team managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.TeamGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.Team{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.TeamGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				util:  &connectorHelper{}}),
			managed.WithCreationGracePeriod(10*time.Second),
			managed.WithInitializers(managed.NewDefaultProviderConfig(mgr.GetClient())),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

// TODO(lb): rename this poorly named interface
type Connector interface {
	GetService(repo service.Repository) teamsvc.Service
	ExtractCredentials(context.Context, v1.CredentialsSource, client.Client, v1.CommonCredentialSelectors) ([]byte, error)
}

type connectorHelper struct{}

func (c *connectorHelper) GetService(repo service.Repository) teamsvc.Service {
	return teamsvc.NewService(repo)
}

func (c *connectorHelper) ExtractCredentials(ctx context.Context, source v1.CredentialsSource, kube client.Client, selector v1.CommonCredentialSelectors) ([]byte, error) {
	return resource.CommonCredentialExtractor(ctx, source, kube, selector)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
	util  Connector
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Team)
	if !ok {
		return nil, errors.New(errNotTeam)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.New(errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	// TODO(liambaker): this will not extract stringData from configMap
	cd := pc.Spec.Credentials
	data, err := c.util.ExtractCredentials(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	var service teamsvc.Service
	switch pc.Spec.Storage.Type {
	case "neo4j":
		store, err := storage.NewNeo4jStorage(data)
		if err != nil {
			return nil, errors.Wrap(err, "client")
		}
		service = teamsvc.NewService(store)
	default:
		return nil, errNewService
	}

	return &external{service: service, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube    client.Client
	service teamsvc.Service
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Team)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTeam)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	currentParams := cr.Spec.ForProvider.DeepCopy()

	resp, err := e.service.GetTeam(meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot get team")
	}

	cr.Status.AtProvider = generateTeamObservation(resp)

	switch cr.Status.AtProvider.Status {
	case string(storetypes.StatusAvailable):
		cr.SetConditions(v1.Available())
	case string(storetypes.StatusUnavailable):
		cr.SetConditions(v1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: cmp.Equal(currentParams.Members, resp.Members) && cmp.Equal(currentParams.ManagedBy.User, resp.ManagedBy),
		Diff:             cmp.Diff(currentParams.Members, resp.Members) + cmp.Diff(currentParams.ManagedBy.User, resp.ManagedBy),
	}, err
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Team)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTeam)
	}

	params := cr.Spec.ForProvider.DeepCopy()

	cr.SetConditions(v1.Creating())
	uuid, err := e.service.CreateTeam(params)

	return postCreate(cr, managed.ExternalCreation{ExternalNameAssigned: true}, uuid, err)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Team)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTeam)
	}

	params := cr.Spec.ForProvider.DeepCopy()

	err := e.service.UpdateTeam(meta.GetExternalName(cr), params)

	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update team")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Team)
	if !ok {
		return errors.New(errNotTeam)
	}

	cr.SetConditions(v1.Deleting())
	err := e.service.DeleteTeam(meta.GetExternalName(cr))

	return errors.Wrap(resource.Ignore(storetypes.IsEntityNotFoundNeo4jErr, err), "cannot delete team")
}

func generateTeamObservation(r *svctypes.GetTeamResponse) v1alpha1.TeamObservation {
	return v1alpha1.TeamObservation{
		NodeID: r.NodeID,
		Status: string(r.Status),
	}
}

func postCreate(cr *v1alpha1.Team, ec managed.ExternalCreation, uuid string, err error) (managed.ExternalCreation, error) {
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create team")
	}

	meta.SetExternalName(cr, uuid)
	return ec, nil
}
