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
	"testing"

	"github.com/pkg/errors"

	"github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	"github.com/VariableExp0rt/powerbroker/internal/service"
	svctypes "github.com/VariableExp0rt/powerbroker/internal/service/types"
	storetypes "github.com/VariableExp0rt/powerbroker/internal/storage/types"
	"github.com/google/go-cmp/cmp"

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
	teamUuid          = "d48c43d0-cff9-4a9b-8971-5ed33b46ea84"
	manager           = "bowser"
	members           = []string{"wario", "toad", "princess"}
	personas          = []string{"entry-to-bowser-castle-role"}
	errInternalServer = errors.New("internal server error")
)

type teamModifier func(*v1alpha1.Team)

func withConditions(c ...v1.Condition) teamModifier {
	return func(p *v1alpha1.Team) {
		p.SetConditions(c...)
	}
}

func withExternalName(uuid string) teamModifier {
	return func(p *v1alpha1.Team) {
		meta.SetExternalName(p, uuid)
	}
}

func withStatus(o v1alpha1.TeamObservation) teamModifier {
	return func(p *v1alpha1.Team) {
		p.Status.AtProvider = o
	}
}

func withSpec(sp v1alpha1.TeamParameters) teamModifier {
	return func(p *v1alpha1.Team) {
		p.Spec.ForProvider = sp
	}
}

func team(opts ...teamModifier) *v1alpha1.Team {
	t := &v1alpha1.Team{}
	for _, o := range opts {
		o(t)
	}

	return t
}

type args struct {
	kube       kclient.Client
	repository service.Repository
	cr         *v1alpha1.Team
}

func TestObserve(t *testing.T) {
	type want struct {
		cr  *v1alpha1.Team
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
					MockUpdate: test.NewMockClient().MockUpdate,
				},
				repository: &service.MockRepository{
					MockGetTeam: func(s string) (*svctypes.GetTeamResponse, error) {
						return &svctypes.GetTeamResponse{
							ManagedBy: "bowser",
							Members:   []string{"wario", "toad", "princess"},
							Personas:  []string{"entry-to-bowser-castle-role"},
							NodeID:    teamUuid,
							Status:    "available",
						}, nil
					},
				},
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withConditions(v1.Available()),
					withStatus(v1alpha1.TeamObservation{
						NodeID: teamUuid,
						Status: string(storetypes.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
		"FailedWithDiff": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetTeam: func(s string) (*svctypes.GetTeamResponse, error) {
						return &svctypes.GetTeamResponse{
							ManagedBy: "luigi",
							Members:   []string{"wario", "toad", "princess"},
							Personas:  []string{"entry-to-bowser-castle-role"},
							NodeID:    teamUuid,
							Status:    "available",
						}, nil
					},
				},
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withConditions(v1.Available()),
					withStatus(v1alpha1.TeamObservation{
						NodeID: teamUuid,
						Status: string(storetypes.StatusAvailable),
					}),
				),
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
					Diff: cmp.Diff(
						manager,
						"luigi",
					),
				},
			},
		},
		"FailedWithError": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockGetTeam: func(s string) (*svctypes.GetTeamResponse, error) {
						return &svctypes.GetTeamResponse{}, errInternalServer
					},
				},
				cr: team(
					withExternalName(teamUuid),
				),
			},
			want: want{
				cr: team(
					withExternalName(teamUuid),
				),
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errInternalServer, "cannot get team"),
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
		cr  *v1alpha1.Team
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
				repository: &service.MockRepository{
					MockCreateTeam: func(tp *v1alpha1.TeamParameters) (string, error) {
						return teamUuid, nil
					},
				},
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withExternalName(teamUuid),
					withConditions(v1.Creating()),
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
				o:   managed.ExternalCreation{ExternalNameAssigned: true},
				err: nil,
			},
		},
		"CreateFailed": {
			args: args{
				kube: &test.MockClient{
					MockUpdate: test.NewMockClient().Update,
				},
				repository: &service.MockRepository{
					MockCreateTeam: func(tp *v1alpha1.TeamParameters) (string, error) {
						return "", errInternalServer
					},
				},
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "super-mario-team",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withConditions(v1.Creating()),
				),
				err: errors.Wrap(errInternalServer, "cannot create team"),
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
		cr  *v1alpha1.Team
		o   managed.ExternalUpdate
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulUpdate": {
			args: args{
				repository: &service.MockRepository{
					MockGetTeam: func(s string) (*svctypes.GetTeamResponse, error) {
						return &svctypes.GetTeamResponse{
							ManagedBy: manager,
							Members:   members,
							Personas:  personas,
							NodeID:    teamUuid,
							Status:    storetypes.StatusAvailable,
						}, nil
					},
					MockUpdateTeam: func(s string, tp *v1alpha1.TeamParameters) error {
						return nil
					},
				},
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "justice-league",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "justice-league",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
				err: nil,
			},
		},
		"UpdateFailed": {
			args: args{
				repository: &service.MockRepository{
					MockGetTeam: func(s string) (*svctypes.GetTeamResponse, error) {
						return &svctypes.GetTeamResponse{
							ManagedBy: manager,
							Members:   members,
							Personas:  personas,
							NodeID:    teamUuid,
							Status:    storetypes.StatusAvailable,
						}, nil
					},
					MockUpdateTeam: func(s string, tp *v1alpha1.TeamParameters) error {
						return errInternalServer
					},
				},
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "justice-league",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withSpec(v1alpha1.TeamParameters{
						Name: "justice-league",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
				err: errors.Wrap(errInternalServer, "cannot update team"),
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
		cr  *v1alpha1.Team
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"SuccessfulDelete": {
			args: args{
				repository: &service.MockRepository{
					MockDeleteTeam: func(s string) error {
						return nil
					},
				},
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "something-something",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "something-something",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withConditions(v1.Deleting()),
				),
				err: nil,
			},
		},
		"DeleteFailed": {
			args: args{
				repository: &service.MockRepository{
					MockDeleteTeam: func(s string) error {
						return errInternalServer
					},
				},
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "something-something",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
				),
			},
			want: want{
				cr: team(
					withExternalName(teamUuid),
					withSpec(v1alpha1.TeamParameters{
						Name: "something-something",
						ManagedBy: v1alpha1.ManagedByParameters{
							User: manager,
						},
						Members:  members,
						Personas: personas,
					}),
					withConditions(v1.Deleting()),
				),
				err: errors.Wrap(errInternalServer, "cannot delete team"),
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
