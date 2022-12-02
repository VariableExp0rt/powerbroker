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
	"testing"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service"
	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	"github.com/VariableExp0rt/powerbroker/internal/storage/types"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
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

var (
	externalName      = "f400d302-8e9e-4dd3-af33-63db7f570528"
	personaName       = "my-over-privileged-persona"
	permissionSetRefs = []string{"super-admin-customer1-001", "god-mode-customer2-002"}
	errInternalServer = &types.InternalError{}
)

type personaModifier func(*v1alpha1.Persona)

func withConditions(c ...v1.Condition) personaModifier {
	return func(p *v1alpha1.Persona) {
		p.SetConditions(c...)
	}
}

func withExternalName(uuid string) personaModifier {
	return func(p *v1alpha1.Persona) {
		meta.SetExternalName(p, uuid)
	}
}

func withStatus(o v1alpha1.PersonaObservation) personaModifier {
	return func(p *v1alpha1.Persona) {
		p.Status.AtProvider = o
	}
}

func withSpec(sp v1alpha1.PersonaParameters) personaModifier {
	return func(p *v1alpha1.Persona) {
		p.Spec.ForProvider = sp
	}
}

func persona(opts ...personaModifier) *v1alpha1.Persona {
	p := &v1alpha1.Persona{}
	for _, o := range opts {
		o(p)
	}

	return p
}

type args struct {
	kube       kclient.Client
	repository service.Repository
	cr         *v1alpha1.Persona
}

func TestObserve(t *testing.T) {
	type want struct {
		cr  *v1alpha1.Persona
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
				repository: &service.MockRepository{
					MockGetPersona: func(uuid string) (*svctypes.GetPersonaResponse, error) {
						return &svctypes.GetPersonaResponse{
							References: permissionSetRefs,
							NodeID:     uuid,
							Status:     "available",
						}, nil
					},
				},
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					})),
			},
			want: want{
				cr: persona(
					withConditions(v1.Available()),
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					}),
					withStatus(v1alpha1.PersonaObservation{
						NodeID: externalName,
						Status: string(types.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		"GetFailedDiff": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetPersona: func(uuid string) (*svctypes.GetPersonaResponse, error) {
						return &svctypes.GetPersonaResponse{
							References: []string{"super-admin-customer1-001"},
							NodeID:     uuid,
							Status:     "available",
						}, nil
					},
				},
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					})),
			},
			want: want{
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					}),
					withConditions(v1.Available()),
					withStatus(v1alpha1.PersonaObservation{
						NodeID: externalName,
						Status: string(types.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
					Diff: cmp.Diff(
						permissionSetRefs,
						[]string{"super-admin-customer1-001"},
					),
				},
				err: nil,
			},
		},
		"GetFailedInternalError": {
			args: args{
				cr: persona(
					withExternalName(externalName),
					withConditions(v1.Unavailable()),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					}),
				),
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetPersona: func(uuid string) (*svctypes.GetPersonaResponse, error) {
						return nil, errInternalServer
					},
				},
			},
			want: want{
				cr: persona(
					withExternalName(externalName),
					withConditions(v1.Unavailable()),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: permissionSetRefs,
					}),
				),
				err: errors.Wrap(errInternalServer, "cannot get persona"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.args.repository}
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
		cr  *v1alpha1.Persona
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulCreate": {
			args: args{
				cr: persona(
					withSpec(v1alpha1.PersonaParameters{
						PermissionSets: permissionSetRefs,
						Name:           personaName,
					}),
				),
				repository: &service.MockRepository{
					MockCreatePersona: func(personaName string, permissionSetRefs []string) (string, error) {
						return externalName, nil
					},
				},
			},
			want: want{
				cr: persona(
					withConditions(v1.Creating()),
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						PermissionSets: permissionSetRefs,
						Name:           personaName,
					}),
				),
				o:   managed.ExternalCreation{ExternalNameAssigned: true},
				err: nil,
			},
		},
		"CreateFailed": {
			args: args{
				cr: persona(
					withSpec(v1alpha1.PersonaParameters{
						PermissionSets: permissionSetRefs,
						Name:           personaName,
					}),
				),
				repository: &service.MockRepository{
					MockCreatePersona: func(personaName string, permissionSetRefs []string) (string, error) {
						return "", errInternalServer
					},
				},
			},
			want: want{
				cr: persona(
					withConditions(v1.Creating()),
					withSpec(v1alpha1.PersonaParameters{
						PermissionSets: permissionSetRefs,
						Name:           personaName,
					}),
				),
				err: errors.Wrap(errInternalServer, "cannot create persona"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.args.repository}
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
		cr  *v1alpha1.Persona
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulUpdate": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetPersona: func(uuid string) (*svctypes.GetPersonaResponse, error) {
						return &svctypes.GetPersonaResponse{
							References: append(permissionSetRefs, "some-new-persona"),
							NodeID:     uuid,
							Status:     "available",
						}, nil
					},
					MockUpdatePersona: func(personaName, personaUuid string, permissionSetUuids []string) error {
						return nil
					},
				},
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: append(permissionSetRefs, "some-new-persona"),
					})),
			},
			want: want{
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: append(permissionSetRefs, "some-new-persona"),
					})),
				err: nil,
			},
		},
		"UpdateFailed": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetPersona: func(uuid string) (*svctypes.GetPersonaResponse, error) {
						return &svctypes.GetPersonaResponse{
							References: permissionSetRefs,
							NodeID:     uuid,
							Status:     "unavailable",
						}, nil
					},
					MockUpdatePersona: func(personaName, personaUuid string, permissionSetUuids []string) error {
						return errInternalServer
					},
				},
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: append(permissionSetRefs, "some-new-persona"),
					})),
			},
			want: want{
				cr: persona(
					withExternalName(externalName),
					withSpec(v1alpha1.PersonaParameters{
						Name:           personaName,
						PermissionSets: append(permissionSetRefs, "some-new-persona"),
					})),
				err: errors.Wrap(errInternalServer, "cannot update persona"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.args.repository}
			_, err := e.Update(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.cr, tc.args.cr, test.EquateConditions()); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type want struct {
		cr  *v1alpha1.Persona
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulDelete": {
			args: args{
				cr: persona(),
				repository: &service.MockRepository{
					MockDeletePersona: func(personaUuid string) error {
						return nil
					},
				},
			},
			want: want{
				cr:  persona(withConditions(v1.Deleting())),
				err: nil,
			},
		},
		"DeleteFailed": {
			args: args{
				cr: persona(),
				repository: &service.MockRepository{
					MockDeletePersona: func(personaUuid string) error {
						return errInternalServer
					},
				},
			},
			want: want{
				cr:  persona(withConditions(v1.Deleting())),
				err: errors.Wrap(errInternalServer, "cannot delete persona"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.args.repository}
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
