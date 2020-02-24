package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"os"
)

func newSession(env environ) (*session.Session, error) {
	// Check for the AWS_DEFAULT_REGION
	if region := env.Getenv("AWS_DEFAULT_REGION"); region != "" {
		env.Setenv("AWS_REGION", region)
	}
	sess := session.Must(session.NewSession())
	if len(aws.StringValue(sess.Config.Region)) == 0 {
		meta := ec2metadata.New(sess)
		identity, err := meta.GetInstanceIdentityDocument()
		if err != nil {
			return nil, err
		}
		return session.NewSession(&aws.Config{
			Region: aws.String(identity.Region),
		})
	}
	return sess, nil
}

func getSSMParams(client ssmClient, names []*string, decrypt bool, nofail bool) (map[string]string, error) {
	input := &ssm.GetParametersInput{
		WithDecryption: aws.Bool(decrypt),
		Names:          names,
	}
	resp, err := client.GetParameters(input)
	if err != nil && !nofail {
		return nil, err
	}

	if len(resp.InvalidParameters) > 0 {
		if !nofail {
			return nil, newInvalidParametersError(resp)
		}
		fmt.Fprintf(os.Stderr, "ssm-env: %v\n", newInvalidParametersError(resp))
	}

	values := make(map[string]string)

	for _, p := range resp.Parameters {
		values[*p.Name] = *p.Value
	}

	return values, nil
}
