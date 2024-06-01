package source

import (
	"os"
	"strings"

	"github.com/slack-go/slack"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

type slackNotifier struct {
	api     ISlackClient
	channel string
	owner   string
}

type ISlackClient interface {
	PostMessage(channel string, options ...slack.MsgOption) (string, string, error)
}

type slackClient struct {
	api *slack.Client
}

func (s slackClient) PostMessage(channel string, options ...slack.MsgOption) (string, string, error) {
	return s.api.PostMessage(channel, options...)
}

func NewSlackNotifier(owner string) *slackNotifier {
	token := os.Getenv("SLACK_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL")
	api := slackClient{api: slack.New(token)}
	return &slackNotifier{api: api, channel: channel, owner: owner}
}

func (s slackNotifier) NotifyChanges(changes *plan.Changes) error {
	if len(changes.Create) == 0 && len(changes.Delete) == 0 && len(changes.UpdateNew) == 0 {
		return nil
	}
	messages := []string{}
	for _, endpoint := range changes.Create {
		for _, target := range endpoint.Targets {
			messages = append(messages, "Create: "+endpoint.DNSName+" -> "+target)
		}
	}

	for i, desired := range changes.UpdateNew {
		current := changes.UpdateOld[i]

		add, remove, _ := provider.Difference(current.Targets, desired.Targets)
		for _, a := range add {
			messages = append(messages, "Create: "+current.DNSName+" -> "+a)
		}
		for _, a := range remove {
			messages = append(messages, "Delete: "+current.DNSName+" -> "+a)
		}
	}

	for _, endpoint := range changes.Delete {
		for _, target := range endpoint.Targets {
			messages = append(messages, "Delete: "+endpoint.DNSName+" -> "+target)
		}
	}
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*DNS Configured Successfully!*", false, false),
			nil, nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", strings.Join(messages, "\n"), false, false),
			nil, nil,
		),
		slack.NewRichTextBlock("Owner",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement("Owner", &slack.RichTextSectionTextStyle{Bold: true}),
				slack.NewRichTextSectionTextElement(": "+s.owner, nil),
			),
		),
	}
	attachment := slack.Attachment{
		Color: "#36D399",
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
	}
	_, _, err := s.api.PostMessage(s.channel, slack.MsgOptionAttachments(attachment))

	return err
}

func (s slackNotifier) NotifyFail(changes *plan.Changes, errInput error) error {
	if len(changes.Create) == 0 && len(changes.Delete) == 0 && len(changes.UpdateNew) == 0 {
		return nil
	}

	messages := []string{}
	for _, endpoint := range changes.Create {
		for _, target := range endpoint.Targets {
			messages = append(messages, "Create: "+endpoint.DNSName+" -> "+target)
		}
	}

	for i, desired := range changes.UpdateNew {
		current := changes.UpdateOld[i]

		add, remove, _ := provider.Difference(current.Targets, desired.Targets)
		for _, a := range add {
			messages = append(messages, "Create: "+current.DNSName+" -> "+a)
		}
		for _, a := range remove {
			messages = append(messages, "Delete: "+current.DNSName+" -> "+a)
		}
	}

	for _, endpoint := range changes.Delete {
		for _, target := range endpoint.Targets {
			messages = append(messages, "Delete: "+endpoint.DNSName+" -> "+target)
		}
	}
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*DNS Configuration Failed.*", false, false),
			nil, nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", strings.Join(messages, "\n"), false, false),
			nil, nil,
		),
		slack.NewRichTextBlock("Owner",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement("Owner", &slack.RichTextSectionTextStyle{Bold: true}),
				slack.NewRichTextSectionTextElement(": "+s.owner, nil),
			),
		),
	}
	attachment := slack.Attachment{
		Color: "#a30200",
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
	}
	_, _, err := s.api.PostMessage(s.channel, slack.MsgOptionAttachments(attachment))

	return err
}
