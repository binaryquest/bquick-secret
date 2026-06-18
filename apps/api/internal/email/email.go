package email

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type Sender interface {
	SendSecretLink(ctx context.Context, recipientEmail, secretURL string) error
	SendRevealNotice(ctx context.Context, senderEmail, secretURL string) error
}

type Disabled struct{}

func (Disabled) SendSecretLink(context.Context, string, string) error {
	return errors.New("email disabled")
}

func (Disabled) SendRevealNotice(context.Context, string, string) error {
	return errors.New("email disabled")
}

type SES struct {
	client *sesv2.Client
	from   string
}

func NewSES(ctx context.Context, region, from string) (Sender, error) {
	if strings.TrimSpace(region) == "" || strings.TrimSpace(from) == "" {
		return nil, errors.New("missing SES configuration")
	}
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &SES{client: sesv2.NewFromConfig(cfg), from: from}, nil
}

func (s *SES) SendSecretLink(ctx context.Context, recipientEmail, secretURL string) error {
	body := fmt.Sprintf(`Someone shared an encrypted secret with you.

Open the secret page here:
%s

For privacy, this email does not include the browser-only decrypt key. Ask the sender for the full secure link or fragment key through a separate channel.

This secret may only be viewable once and will expire automatically.

The plaintext secret is decrypted in your browser. The server cannot read it.
`, secretURL)

	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination:      &types.Destination{ToAddresses: []string{recipientEmail}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String("You received an encrypted secret"), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")}},
		}},
	})
	return err
}

func (s *SES) SendRevealNotice(ctx context.Context, senderEmail, secretURL string) error {
	body := fmt.Sprintf(`Your bQuick Secret link was opened and the encrypted payload was revealed in the recipient browser.

Secret page:
%s

The server still cannot read the plaintext secret or browser-only decrypt key.
`, secretURL)

	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination:      &types.Destination{ToAddresses: []string{senderEmail}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String("Your bQuick Secret was revealed"), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(body), Charset: aws.String("UTF-8")}},
		}},
	})
	return err
}
