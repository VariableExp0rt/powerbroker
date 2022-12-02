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
	"testing"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service"
	"github.com/VariableExp0rt/powerbroker/internal/service/permissionset"
	"github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage/neo4j/transaction"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

var _ managed.ExternalClient = &external{}
var _ managed.ExternalConnecter = &connector{}
var _ Connector = &connectorHelper{}

var (
	externalName = "712081a1-0da3-46cd-bab3-ee1852723c4f"
	binding      = v1alpha1.AccountRoleBinding{
		Alias:        "my-aws-account-alias",
		Account:      "111111111111",
		AccountClass: "production",
		RoleName:     "my-aws-iam-role",
	}
	errInternalServer = &transaction.InternalError{Message: "internal server error"}
	errSecretNotFound = errors.New("no resource found for secret name")
)

type MockCredentialHelper struct {
	MockGetService         func(repo service.Repository) permissionset.Service
	MockExtractCredentials func(context.Context, v1.CredentialsSource, kclient.Client, v1.CommonCredentialSelectors) ([]byte, error)
}

func (_c *MockCredentialHelper) GetService(repo service.Repository) permissionset.Service {
	return _c.MockGetService(repo)
}

func (_c *MockCredentialHelper) ExtractCredentials(ctx context.Context, source v1.CredentialsSource, kube kclient.Client, selector v1.CommonCredentialSelectors) ([]byte, error) {
	return _c.MockExtractCredentials(ctx, source, kube, selector)
}

type psModifier = func(*v1alpha1.PermissionSet)

func withConditions(cnds ...v1.Condition) psModifier {
	return func(ps *v1alpha1.PermissionSet) {
		ps.SetConditions(cnds...)
	}
}

func withSpec(p v1alpha1.PermissionSetParameters) psModifier {
	return func(ps *v1alpha1.PermissionSet) {
		ps.Spec.ForProvider = p
	}
}

func withStatus(o v1alpha1.PermissionSetObservation) psModifier {
	return func(ps *v1alpha1.PermissionSet) {
		ps.Status.AtProvider = o
	}
}

func withExternalName(uuid string) psModifier {
	return func(ps *v1alpha1.PermissionSet) {
		meta.SetExternalName(ps, uuid)
	}
}

func withProviderConfig(rs v1.Reference) psModifier {
	return func(ps *v1alpha1.PermissionSet) {
		ps.Spec.ProviderConfigReference = &rs
	}
}

func permissionSet(opts ...psModifier) *v1alpha1.PermissionSet {
	ps := &v1alpha1.PermissionSet{}
	for _, o := range opts {
		o(ps)
	}

	return ps
}

type args struct {
	kube    kclient.Client
	cr      *v1alpha1.PermissionSet
	service service.Repository
}

func TestExtractCredentials(t *testing.T) {
	type chArgs struct {
		kube     kclient.Client
		selector v1.CommonCredentialSelectors
		source   v1.CredentialsSource
		util     Connector
	}

	type want struct {
		data []byte
		err  error
	}

	cases := map[string]struct {
		args chArgs
		want want
	}{
		"ExtractCredentialsSuccess": {
			args: chArgs{
				kube: &test.MockClient{
					MockGet: test.NewMockClient().Get,
				},
				source: v1.CredentialsSourceSecret,
				selector: v1.CommonCredentialSelectors{
					SecretRef: &v1.SecretKeySelector{},
				},
				util: &MockCredentialHelper{
					MockExtractCredentials: func(ctx context.Context, cs v1.CredentialsSource, c kclient.Client, ccs v1.CommonCredentialSelectors) ([]byte, error) {
						return []byte("some-secret-data"), nil
					},
				},
			},
			want: want{
				data: []byte("some-secret-data"),
				err:  nil,
			},
		},
		"ExtractCredentialsFailure": {
			args: chArgs{
				kube: &test.MockClient{
					MockGet: test.NewMockClient().Get,
				},
				source: v1.CredentialsSourceSecret,
				selector: v1.CommonCredentialSelectors{
					SecretRef: &v1.SecretKeySelector{},
				},
				util: &MockCredentialHelper{
					MockExtractCredentials: func(ctx context.Context, cs v1.CredentialsSource, c kclient.Client, ccs v1.CommonCredentialSelectors) ([]byte, error) {
						return nil, errSecretNotFound
					},
				},
			},
			want: want{
				data: nil,
				err:  errSecretNotFound,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &connector{util: tc.args.util}
			data, err := c.util.ExtractCredentials(
				context.Background(),
				tc.args.source,
				tc.args.kube,
				tc.args.selector,
			)

			if diff := cmp.Diff(data, tc.want.data); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(err, tc.want.err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	type connectArgs struct {
		cr        *v1alpha1.PermissionSet
		connector connector
	}
	type want struct {
		err error
	}

	cases := map[string]struct {
		args connectArgs
		want want
	}{
		"ConnectFailureGetClient": {
			args: connectArgs{
				cr: permissionSet(
					withProviderConfig(v1.Reference{
						Name: "test-config",
					}),
				),
				connector: connector{
					kube: &test.MockClient{
						MockGet: test.NewMockClient().Get,
					},
					usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error {
						return nil
					}),
					util: &MockCredentialHelper{
						MockGetService: func(repo service.Repository) permissionset.Service {
							return nil
						},
						MockExtractCredentials: func(ctx context.Context, cs v1.CredentialsSource, c kclient.Client, ccs v1.CommonCredentialSelectors) ([]byte, error) {
							return []byte{}, nil
						},
					},
				},
			},
			want: want{
				err: errNewService,
			},
		},
		"ConnectFailureGetCreds": {
			args: connectArgs{
				cr: permissionSet(
					withProviderConfig(v1.Reference{
						Name: "test-config",
					}),
				),
				connector: connector{
					kube: &test.MockClient{
						MockGet: test.NewMockClient().Get,
					},
					usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error {
						return nil
					}),
					util: &MockCredentialHelper{
						MockExtractCredentials: func(ctx context.Context, cs v1.CredentialsSource, c kclient.Client, ccs v1.CommonCredentialSelectors) ([]byte, error) {
							return []byte{}, errSecretNotFound
						},
					},
				},
			},
			want: want{
				err: errors.Wrap(errSecretNotFound, "cannot get credentials"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &connector{kube: tc.args.connector.kube,
				usage: tc.args.connector.usage,
				util:  tc.args.connector.util,
			}
			_, err := c.Connect(context.Background(), tc.args.cr)

			if diff := cmp.Diff(err, tc.want.err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestObserve(t *testing.T) {
	type want struct {
		cr  *v1alpha1.PermissionSet
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulAvailable": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				cr: permissionSet(
					withConditions(v1.Available(), v1.ReconcileSuccess()),
					withExternalName("712081a1-0da3-46cd-bab3-ee1852723c4f"),
					withSpec(v1alpha1.PermissionSetParameters{
						BindTo: binding,
					}),
					withStatus(v1alpha1.PermissionSetObservation{
						NodeID: externalName,
						Status: transaction.StatusAvailable,
					}),
				),
				service: &service.MockRepository{
					MockGetPermissionSet: func(uuid string) (*types.GetPermissionSetResponse, error) {
						return &types.GetPermissionSetResponse{
							Binding: binding,
							Status:  "available",
							NodeID:  "712081a1-0da3-46cd-bab3-ee1852723c4f",
						}, nil
					},
				},
			},
			want: want{
				cr: permissionSet(
					withConditions(v1.Available(), v1.ReconcileSuccess()),
					withExternalName(externalName),
					withStatus(v1alpha1.PermissionSetObservation{
						NodeID: externalName,
						Status: transaction.StatusAvailable,
					}),
					withSpec(v1alpha1.PermissionSetParameters{BindTo: binding}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"GetFailedUnavailable": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				cr: permissionSet(
					withConditions(v1.Unavailable(), v1.ReconcileSuccess()),
					withExternalName("712081a1-0da3-46cd-bab3-ee1852723c4f"),
					withSpec(v1alpha1.PermissionSetParameters{
						BindTo: binding,
					}),
					withStatus(v1alpha1.PermissionSetObservation{
						NodeID: "712081a1-0da3-46cd-bab3-ee1852723c4f",
						Status: "unavailable",
					}),
				),
				service: &service.MockRepository{
					MockGetPermissionSet: func(uuid string) (*types.GetPermissionSetResponse, error) {
						return nil, errInternalServer
					},
				},
			},
			want: want{
				cr: permissionSet(
					withConditions(v1.Unavailable(), v1.ReconcileSuccess()),
					withExternalName(externalName),
					withSpec(v1alpha1.PermissionSetParameters{
						BindTo: binding,
					}),
					withStatus(v1alpha1.PermissionSetObservation{
						NodeID: externalName,
						Status: transaction.StatusUnavailable,
					}),
				),
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errInternalServer, "cannot get permissionset"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{service: tc.args.service}
			o, err := e.Observe(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type want struct {
		cr  *v1alpha1.PermissionSet
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulCreate": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				service: &service.MockRepository{
					MockCreatePermissionSet: func(name string, binding v1alpha1.AccountRoleBinding) (string, error) {
						return "712081a1-0da3-46cd-bab3-ee1852723c4f", nil
					},
				},
				cr: permissionSet(
					withConditions(v1.Creating()),
				),
			},
			want: want{
				cr: permissionSet(
					withExternalName(externalName),
					withConditions(v1.Creating())),
				err: nil,
				o:   managed.ExternalCreation{ExternalNameAssigned: true},
			},
		},
		"CreateFailed": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				cr: permissionSet(withConditions(v1.Creating())),
				service: &service.MockRepository{
					MockCreatePermissionSet: func(name string, binding v1alpha1.AccountRoleBinding) (string, error) {
						return "", errInternalServer
					},
				},
			},
			want: want{
				cr:  permissionSet(withConditions(v1.Creating())),
				o:   managed.ExternalCreation{ExternalNameAssigned: false},
				err: errors.Wrap(errInternalServer, "cannot create permissionset"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{kube: tc.args.kube, service: tc.args.service}
			o, err := e.Create(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	type want struct {
		cr  *v1alpha1.PermissionSet
		o   managed.ExternalUpdate
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulUpdate": {
			args: args{
				service: &service.MockRepository{
					MockGetPermissionSet: func(uuid string) (*types.GetPermissionSetResponse, error) {
						return &types.GetPermissionSetResponse{}, nil
					},
					MockUpdatePermissionSet: func(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error {
						return nil
					},
				},
				cr: permissionSet(withSpec(v1alpha1.PermissionSetParameters{BindTo: binding})),
			},
			want: want{
				cr: permissionSet(withSpec(v1alpha1.PermissionSetParameters{BindTo: binding})),
			},
		},
		"UpdateFailed": {
			args: args{
				service: &service.MockRepository{
					MockGetPermissionSet: func(uuid string) (*types.GetPermissionSetResponse, error) {
						return &types.GetPermissionSetResponse{}, nil
					},
					MockUpdatePermissionSet: func(permissionSetUuid, crName string, binding v1alpha1.AccountRoleBinding) error {
						return errInternalServer
					},
				},
				cr: permissionSet(withSpec(v1alpha1.PermissionSetParameters{BindTo: binding})),
			},
			want: want{
				cr:  permissionSet(withSpec(v1alpha1.PermissionSetParameters{BindTo: binding})),
				err: errors.Wrap(errInternalServer, "cannot update permissionset"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{kube: tc.args.kube, service: tc.args.service}
			o, err := e.Update(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type want struct {
		cr  *v1alpha1.PermissionSet
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulDelete": {
			args: args{
				cr: permissionSet(),
				service: &service.MockRepository{
					MockDeletePermissionSet: func(permissionSetUuid string) error {
						return nil
					},
				},
			},
			want: want{
				cr:  permissionSet(withConditions(v1.Deleting())),
				err: nil,
			},
		},
		"DeleteFailed": {
			args: args{
				cr: permissionSet(),
				service: &service.MockRepository{
					MockDeletePermissionSet: func(permissionSetUuid string) error {
						return errInternalServer
					},
				},
			},
			want: want{
				cr:  permissionSet(withConditions(v1.Deleting())),
				err: errors.Wrap(errInternalServer, "cannot delete permissionset"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{kube: tc.args.kube, service: tc.args.service}
			err := e.Delete(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
