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
	"testing"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service"
	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
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
	externalName      = "af3d69e0-1619-4b7d-af39-4478fab5c9c5"
	userName          = "my-least-privileged-user"
	personaRefs       = []string{"super-admin-smash-bros", "production-access-for-everyone"}
	errInternalServer = &storetypes.InternalError{}
)

type userModifier func(*v1alpha1.User)

func withConditions(c ...v1.Condition) userModifier {
	return func(p *v1alpha1.User) {
		p.SetConditions(c...)
	}
}

func withExternalName(uuid string) userModifier {
	return func(p *v1alpha1.User) {
		meta.SetExternalName(p, uuid)
	}
}

func withStatus(o v1alpha1.UserObservation) userModifier {
	return func(p *v1alpha1.User) {
		p.Status.AtProvider = o
	}
}

func withSpec(sp v1alpha1.UserParameters) userModifier {
	return func(p *v1alpha1.User) {
		p.Spec.ForProvider = sp
	}
}

func user(opts ...userModifier) *v1alpha1.User {
	u := &v1alpha1.User{}
	for _, o := range opts {
		o(u)
	}

	return u
}

type args struct {
	kube       kclient.Client
	repository service.Repository
	cr         *v1alpha1.User
}

func TestObserve(t *testing.T) {
	type want struct {
		cr  *v1alpha1.User
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
					MockGetUser: func(userUuid string) (*svctypes.GetUserResponse, error) {
						return &svctypes.GetUserResponse{
							NodeID:     externalName,
							References: personaRefs,
							Status:     "available",
						}, nil
					},
				},
				cr: user(
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
				),
			},
			want: want{
				cr: user(
					withConditions(v1.Available()),
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
					withStatus(v1alpha1.UserObservation{
						NodeID: externalName,
						Status: string(storetypes.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
			},
		},
		"GetFailedWithDiff": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetUser: func(userUuid string) (*svctypes.GetUserResponse, error) {
						return &svctypes.GetUserResponse{
							NodeID:     externalName,
							References: personaRefs,
							Status:     "available",
						}, nil
					},
				},
				cr: user(
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "the-scoped-readonly-persona"),
					}),
				),
			},
			want: want{
				cr: user(
					withConditions(v1.Available()),
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "the-scoped-readonly-persona"),
					}),
					withStatus(v1alpha1.UserObservation{
						NodeID: externalName,
						Status: string(storetypes.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists: true,
					ResourceUpToDate: cmp.Equal(
						append(personaRefs, "the-scoped-readonly-persona"),
						personaRefs,
					),
					Diff: cmp.Diff(
						append(personaRefs, "the-scoped-readonly-persona"),
						personaRefs,
					),
				},
			},
		},
		"GetFailedInternalError": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetUser: func(userUuid string) (*svctypes.GetUserResponse, error) {
						return nil, errInternalServer
					},
				},
				cr: user(
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
				),
			},
			want: want{
				cr: user(
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
				),
				err: errors.Wrap(errInternalServer, "cannot get user"),
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
		cr  *v1alpha1.User
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulCreate": {
			args: args{
				repository: &service.MockRepository{
					MockCreateUser: func(userName string, personaRefs []string) (string, error) {
						return externalName, nil
					},
				},
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
				),
			},
			want: want{
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
					withConditions(v1.Creating()),
					withExternalName(externalName),
				),
				o: managed.ExternalCreation{
					ExternalNameAssigned: true,
				},
			},
		},
		"CreateFailed": {
			args: args{
				repository: &service.MockRepository{
					MockCreateUser: func(userName string, personaRefs []string) (string, error) {
						return "", errInternalServer
					},
				},
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
				),
			},
			want: want{
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: personaRefs,
					}),
					withConditions(v1.Creating()),
				),
				err: errors.Wrap(errInternalServer, "cannot create user"),
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
		cr  *v1alpha1.User
		o   managed.ExternalUpdate
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulUpdate": {
			args: args{
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "evil-persona-delete-storage-bkts"),
					}),
					withExternalName(externalName),
				),
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetUser: func(userUuid string) (*svctypes.GetUserResponse, error) {
						return &svctypes.GetUserResponse{
							NodeID:     externalName,
							Status:     "available",
							References: append(personaRefs, "evil-persona-delete-storage-bkts"),
						}, nil
					},
					MockUpdateUser: func(userName, userUuid string, personaRefs []string) error {
						return nil
					},
				},
			},
			want: want{
				cr: user(
					withExternalName(externalName),
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "evil-persona-delete-storage-bkts"),
					}),
				),
				err: nil,
			},
		},
		"UpdateFailed": {
			args: args{
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "evil-persona-delete-storage-bkts"),
					}),
					withExternalName(externalName),
				),
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetUser: func(userUuid string) (*svctypes.GetUserResponse, error) {
						return &svctypes.GetUserResponse{
							NodeID:     externalName,
							Status:     "available",
							References: personaRefs,
						}, nil
					},
					MockUpdateUser: func(userName, userUuid string, personaRefs []string) error {
						return errInternalServer
					},
				},
			},
			want: want{
				cr: user(
					withSpec(v1alpha1.UserParameters{
						Name:     userName,
						Personas: append(personaRefs, "evil-persona-delete-storage-bkts"),
					}),
					withExternalName(externalName),
				),
				err: errors.Wrap(errInternalServer, "cannot update user"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.args.repository}
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
		cr  *v1alpha1.User
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulDelete": {
			args: args{
				cr: user(),
				repository: &service.MockRepository{
					MockDeleteUser: func(userUuid string) error {
						return nil
					},
				},
			},
			want: want{
				cr:  user(withConditions(v1.Deleting())),
				err: nil,
			},
		},
		"DeleteFailed": {
			args: args{
				cr: user(),
				repository: &service.MockRepository{
					MockDeleteUser: func(userUuid string) error {
						return errInternalServer
					},
				},
			},
			want: want{
				cr:  user(withConditions(v1.Deleting())),
				err: errors.Wrap(errInternalServer, "cannot delete user"),
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
