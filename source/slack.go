package source

import (
	"os"
	"strings"

	"github.com/slack-go/slack"
	"sigs.k8s.io/external-dns/plan"
)

type slackNotifier struct {
	api      ISlackClient
	channel  string
	owner    string
	provider string
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

func NewSlackNotifier() *slackNotifier {
	token := os.Getenv("SLACK_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL")
	api := slackClient{api: slack.New(token)}
	return &slackNotifier{api: api, channel: channel}
}

func (s slackNotifier) NotifyChanges(changes *plan.Changes) error {
	if len(changes.Create) == 0 && len(changes.Delete) == 0 && len(changes.UpdateNew) == 0 {
		return nil
	}

	messages := []string{}
	for _, change := range changes.Create {
		messages = append(messages, "Create: "+change.DNSName)
	}
	for _, change := range changes.Delete {
		messages = append(messages, "Delete: "+change.DNSName)
	}
	for _, change := range changes.UpdateNew {
		messages = append(messages, "UpdateNew: "+change.DNSName)
	}
	for _, change := range changes.UpdateOld {
		messages = append(messages, "UpdateOld: "+change.DNSName)
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
		slack.NewRichTextBlock("Provider",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement("Provider", &slack.RichTextSectionTextStyle{Bold: true}),
				slack.NewRichTextSectionTextElement(": "+s.provider, nil),
			),
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

	messages := []string{errInput.Error()}
	for _, change := range changes.Create {
		messages = append(messages, "Create: "+change.DNSName)
	}
	for _, change := range changes.Delete {
		messages = append(messages, "Delete: "+change.DNSName)
	}
	for _, change := range changes.UpdateNew {
		messages = append(messages, "UpdateNew: "+change.DNSName)
	}
	for _, change := range changes.UpdateOld {
		messages = append(messages, "UpdateOld: "+change.DNSName)
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
		slack.NewRichTextBlock("Provider",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement("Provider", &slack.RichTextSectionTextStyle{Bold: true}),
				slack.NewRichTextSectionTextElement(": "+s.provider, nil),
			),
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
