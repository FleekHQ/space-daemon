package email

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type ReplacementData struct {
	SenderUsername string `json:"senderUsername"`
  SenderEmail string `json:"senderEmail"`
  ObjectHref string `json:"objectHref"`
}

func SendEmails(recipients []string, url string, senderUsername string, senderEmail string) error {
	sess, err := session.NewSession(&aws.Config{
		Region:aws.String("us-west-2")},
		// key
		// secret
	)
	
	if err != nil {
		return err
	}

	svc := ses.New(sess)

	rep := &ReplacementData{
		SenderUsername: senderUsername,
		SenderEmail: senderEmail,
		ObjectHref: url,
	}
	b, _ := json.Marshal(rep)
	sb := string(b)

	d := make([]*ses.BulkEmailDestination, len(recipients))
	for _, r := range recipients {
		dest := &ses.BulkEmailDestination{
			Destination: &ses.Destination{
				ToAddresses: []*string{&r},
			},
			ReplacementTemplateData: &sb,
		}
		d = append(d, dest)
	}

	s := "share@space.storage"
	t := "ShareFolder"

	i := &ses.SendBulkTemplatedEmailInput{
		Destinations: d,
		Source: &s,
		Template: &t,
	}
	_, e := svc.SendBulkTemplatedEmail(i)

	return e
}

