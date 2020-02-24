package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"os"

	"strings"
	"text/template"
)

type ssmFetcher struct {
	parsedTemplate *template.Template
	ssm            ssmClient
	os             environ
	batchSize      int
}

func (f *ssmFetcher) expandEnviron(decrypt bool, nofail bool) error {
	params := make(map[string][]string)
	for _, env := range f.os.Environ() {
		name, value := splitVar(env)
		param, err := execTemplate(f.parsedTemplate, name, value)
		if err != nil {
			return fmt.Errorf("determining name of parameter: %v", err)
		}

		if param != "" {
			if _, ok := params[param]; !ok {
				params[param] = []string{name}
			} else {
				params[param] = append(params[param], name)
			}
		}
	}

	names := make([]*string, 0, len(params))
	for name, _ := range params {
		names = append(names, aws.String(name))
	}

	for start := 0; start < len(names); start += f.batchSize {
		end := start + f.batchSize
		if end > len(names) {
			end = len(names)
		}
		values, err := getSSMParams(f.ssm, names[start:end], decrypt, nofail)
		if err != nil {
			return err
		}

		for name, envs := range params {
			if val, ok := values[name]; ok {
				for _, env := range envs {
					f.os.Setenv(env, val)
				}
			}
		}
	}

	return nil
}

type ssmClient interface {
	GetParameters(*ssm.GetParametersInput) (*ssm.GetParametersOutput, error)
}

type environ interface {
	Environ() []string
	Setenv(key, value string)
	Getenv(key string) string
}

type osEnviron int

func (e osEnviron) Environ() []string {
	return os.Environ()
}

func (e osEnviron) Setenv(key, val string) {
	_ = os.Setenv(key, val)
}

func (e osEnviron) Getenv(key string) string {
	return os.Getenv(key)
}

func splitVar(name string) (key, value string) {
	splitted := strings.Split(name, "=")
	return splitted[0], splitted[1]
}

type invalidParametersError struct {
	InvalidParameters []string
}

func newInvalidParametersError(resp *ssm.GetParametersOutput) *invalidParametersError {
	e := new(invalidParametersError)
	for _, p := range resp.InvalidParameters {
		if p == nil {
			continue
		}

		e.InvalidParameters = append(e.InvalidParameters, *p)
	}
	return e
}

func (e *invalidParametersError) Error() string {
	return fmt.Sprintf("invalid parameters: %v", e.InvalidParameters)
}
