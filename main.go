package main

import (
	"bytes"
	"flag"
	"fmt"

	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const DefaultTemplate = `{{ if hasPrefix .Value "ssm://" }}{{ trimPrefix .Value "ssm://" }}{{ end }}`

var (
	templateText = flag.String("template", DefaultTemplate, "The template used to determine what "+
		"the SSM parameter name is for an environment variable. When this template returns an empty string, "+
		"the env variable is not an SSM parameter")
	decrypt = flag.Bool("with-decryption", false, "Will attempt to decrypt the parameter, "+
		"and set the env var as plaintext")
	nofail = flag.Bool("no-fail", false, "Don't fail if error retrieving parameter")

	// Array Members: Minimum number of 1 item. Maximum number of 10 items.
	// https://docs.aws.amazon.com/systems-manager/latest/APIReference/API_GetParameters.html#API_GetParameters_RequestSyntax
	batchSize = flag.Int("batch-size", 10, "Batch size")
)

var templateFuncs = template.FuncMap{
	"contains":   strings.Contains,
	"hasPrefix":  strings.HasPrefix,
	"hasSuffix":  strings.HasSuffix,
	"trimPrefix": strings.TrimPrefix,
	"trimSuffix": strings.TrimSuffix,
	"trimSpace":  strings.TrimSpace,
	"trimLeft":   strings.TrimLeft,
	"trimRight":  strings.TrimRight,
	"trim":       strings.Trim,
	"title":      strings.Title,
	"toTitle":    strings.ToTitle,
	"toLower":    strings.ToLower,
	"toUpper":    strings.ToUpper,
}

func main() {
	var environ osEnviron

	args := parseFlags()
	cmdPath, err := exec.LookPath(args[0])
	must(err)

	fetcher := &ssmFetcher{
		parsedTemplate: parseTemplate(*templateText),
		ssm:            ssm.New(session.Must(newSession(environ))),
		os:             environ,
		batchSize:      *batchSize,
	}

	must(fetcher.expandEnviron(*decrypt, *nofail))
	must(syscall.Exec(cmdPath, args[0:], os.Environ()))
}

func parseFlags() []string {
	flag.Parse()
	args := flag.Args()

	if len(args) <= 0 {
		flag.Usage()
		os.Exit(1)
	}

	return args
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssm-env: %v\n", err)
		os.Exit(1)
	}
}

func parseTemplate(templateText string) *template.Template {
	return template.Must(template.New("template").Funcs(templateFuncs).Parse(templateText))
}

func execTemplate(t *template.Template, name, value string) (string, error) {
	b := new(bytes.Buffer)
	if err := t.Execute(b, struct{ Name, Value string }{name, value}); err != nil {
		return "", err
	}

	return b.String(), nil
}
