package template

import (
	"bytes"
	"html/template"

	"github.com/Masterminds/sprig"

	"github.com/pkg/errors"
)

func Render(data interface{}, tmpl string) (string, error) {
	var t, err = defaultTemplate("").Parse(tmpl)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse new template")
	}
	var r bytes.Buffer
	if err = t.Execute(&r, data); err != nil {
		return "", errors.Wrap(err, "failed to render template with data")
	}
	return r.String(), nil
}

func TryRender(data interface{}, tmpl string) string {
	var r, _ = Render(data, tmpl)
	return r
}

func defaultTemplate(name string) *template.Template {
	return template.New(name).Funcs(sprig.FuncMap())
}
