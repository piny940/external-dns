package source

import (
	"fmt"
	"testing"

	"github.com/slack-go/slack"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

var changesSuits = []struct {
	name    string
	changes *plan.Changes
	wantErr bool
}{
	{
		name:    "no changes",
		changes: &plan.Changes{},
		wantErr: false,
	},
	{
		name: "create",
		changes: &plan.Changes{
			Create: []*endpoint.Endpoint{
				{
					DNSName:    "create.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			UpdateOld: []*endpoint.Endpoint{},
			UpdateNew: []*endpoint.Endpoint{},
			Delete:    []*endpoint.Endpoint{},
		},
		wantErr: false,
	},
	{
		name: "delete",
		changes: &plan.Changes{
			Delete: []*endpoint.Endpoint{
				{
					DNSName:    "delete.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			Create:    []*endpoint.Endpoint{},
			UpdateOld: []*endpoint.Endpoint{},
			UpdateNew: []*endpoint.Endpoint{},
		},
		wantErr: false,
	},
	{
		name: "updateNew",
		changes: &plan.Changes{
			UpdateNew: []*endpoint.Endpoint{
				{
					DNSName:    "updateNew.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			Create:    []*endpoint.Endpoint{},
			UpdateOld: []*endpoint.Endpoint{},
			Delete:    []*endpoint.Endpoint{},
		},
		wantErr: false,
	},
	{
		name: "updateOld",
		changes: &plan.Changes{
			UpdateOld: []*endpoint.Endpoint{
				{
					DNSName:    "updateOld.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			Create:    []*endpoint.Endpoint{},
			UpdateNew: []*endpoint.Endpoint{},
			Delete:    []*endpoint.Endpoint{},
		},
		wantErr: false,
	},
	{
		name: "all changes",
		changes: &plan.Changes{
			Create: []*endpoint.Endpoint{
				{
					DNSName:    "create.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			Delete: []*endpoint.Endpoint{
				{
					DNSName:    "delete.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			UpdateNew: []*endpoint.Endpoint{
				{
					DNSName:    "updateNew.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
			UpdateOld: []*endpoint.Endpoint{
				{
					DNSName:    "updateOld.example.com",
					RecordType: "A",
					Targets:    endpoint.Targets{""},
				},
			},
		},
		wantErr: false,
	},
}

type slackClientMock struct{}

func (s slackClientMock) PostMessage(channel string, options ...slack.MsgOption) (string, string, error) {
	fmt.Println("PostMessage to channel: ", channel)
	return "", "", nil
}

func TestNotifyChanges(t *testing.T) {
	t.Parallel()
	for _, tt := range changesSuits[:2] {
		t.Run(tt.name, func(t *testing.T) {
			s := slackNotifier{
				api:     slackClientMock{},
				channel: "C06CB9RUP3J",
			}
			if err := s.NotifyChanges(tt.changes); (err != nil) != tt.wantErr {
				t.Errorf("NotifyChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotifyFail(t *testing.T) {
	t.Parallel()
	for _, tt := range changesSuits[:2] {
		t.Run(tt.name, func(t *testing.T) {
			s := slackNotifier{
				api:     slackClientMock{},
				channel: "C06CB9RUP3J",
			}
			if err := s.NotifyFail(tt.changes, fmt.Errorf("failed...")); (err != nil) != tt.wantErr {
				t.Errorf("NotifyChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
