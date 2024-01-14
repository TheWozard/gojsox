package gohtmx

import (
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/TheWozard/gohtmx/v2/core"
)

type Variable struct {
	Name string
	Func func(*http.Request) any
}

func (tv Variable) Init(f *Framework, w io.Writer) error {
	if !f.CanTemplate() {
		return ErrCannotTemplate
	}
	id := f.Generator.NewFunctionID(tv.Func)
	f.Template = f.Template.Funcs(template.FuncMap{id: tv.Func})
	return Raw(fmt.Sprintf("{{%s := %s $r}}", tv.Name, id)).Init(f, w)
}

type With struct {
	Func    func(*http.Request) core.TemplateData
	Content Component
}

func (t With) Init(f *Framework, w io.Writer) error {
	if !f.CanTemplate() {
		return ErrCannotTemplate
	}
	if t.Content == nil {
		return nil
	}
	id := f.Generator.NewFunctionID(t.Func)
	f.Template = f.Template.Funcs(template.FuncMap{id: t.Func})
	err := Raw(fmt.Sprintf("{{with %s $r}}", id)).Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write with prefix: %w", err)
	}
	err = t.Content.Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write with content: %w", err)
	}
	err = Raw("{{end}}").Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write with suffix: %w", err)
	}
	return nil
}

type Condition struct {
	Condition func(r *http.Request) bool
	Content   Component
}

func (tc Condition) ConditionString(id string) string {
	return fmt.Sprintf("if %s $r", id)
}

func (tc Condition) Init(f *Framework, w io.Writer) error {
	if !f.CanTemplate() {
		return ErrCannotTemplate
	}
	if tc.Content == nil {
		return nil
	}
	// If there is no condition, then just write the content.
	if tc.Condition == nil {
		return tc.Content.Init(f, w)
	}
	id := f.Generator.NewFunctionID(tc.Condition)
	f.Template = f.Template.Funcs(template.FuncMap{
		id: tc.Condition,
	})
	err := Raw("{{"+tc.ConditionString(id)+"}}").Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write condition prefix: %w", err)
	}
	err = tc.Content.Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write condition content: %w", err)
	}
	err = Raw("{{end}}").Init(f, w)
	if err != nil {
		return fmt.Errorf("failed to write condition suffix: %w", err)
	}
	return nil
}

type Conditions []Condition

func (tcs Conditions) Init(f *Framework, w io.Writer) error {
	if !f.CanTemplate() {
		return ErrCannotTemplate
	}
	if len(tcs) == 0 {
		return nil
	}
	if len(tcs) == 1 {
		return tcs[0].Init(f, w)
	}
	elses := Fragment{}
	branches := 0
	for i, tc := range tcs {
		if tc.Condition == nil {
			if tc.Content != nil {
				elses = append(elses, tc.Content)
			}
			continue
		}
		prefix := ""
		if branches > 0 {
			prefix = "else "
		}
		id := f.Generator.NewFunctionID(tc.Condition)
		f.Template = f.Template.Funcs(template.FuncMap{
			id: tc.Condition,
		})
		if err := Raw("{{"+prefix+tc.ConditionString(id)+"}}").Init(f, w); err != nil {
			return fmt.Errorf(`failed to write Conditions[%d]: %w`, i, err)
		}
		if err := tc.Content.Init(f, w); err != nil {
			return fmt.Errorf(`failed to write Conditions[%d].Content: %w`, i, err)
		}
		branches++
	}

	if len(elses) > 0 {
		// If some conditions were written, then we need to write the else condition.
		if branches > 0 {
			if err := Raw("{{else}}").Init(f, w); err != nil {
				return fmt.Errorf("failed to write Conditions else: %w", err)
			}
		}
		elses.Init(f, w)
	}

	if branches > 0 {
		if err := Raw("{{end}}").Init(f, w); err != nil {
			return fmt.Errorf("failed to write Conditions termination: %w", err)
		}
	}
	return nil
}
