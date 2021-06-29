package hook

import (
	"github.com/rs/zerolog/log"
	"html/template"
	"os"

	"net/http"
	"path"
)

// This stuff is used in the templates.
type baseTemplateSettings struct {
	MobileFriendly bool
	DarkMode       bool
	PageName       string
	Arguments      interface{}
}

func makeBaseTemplateSettings(mobileFriendly bool, darkMode bool, pageName string, arguments interface{}) baseTemplateSettings {
	return baseTemplateSettings{mobileFriendly, darkMode, pageName, arguments}
}

func (s *Server) prepareBaseTemplate(t *template.Template) (*template.Template, error) {
	return t.Funcs(map[string]interface{}{
		"settings": makeBaseTemplateSettings,
		//"branding":         getConcreteBrandingFunction(cfg),
		//"sections":         getConcreteSectionFunction(o),
		"PageName": func() string {return "base"},
		//"csrfToken":        func() string { return csrfToken },
	}).ParseFiles(path.Join(s.Config.TemplateDir, "base.html"))
}

func (s *Server) handleSimpleTemplate(templateName string, param interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := template.New(templateName) // the name matters, and must match the filename.
		t, err := s.prepareBaseTemplate(t)
		if err != nil {
			log.Error().Msgf(err.Error())
			http.Error(w, "error preparing base template", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		http.SetCookie(w,&http.Cookie{

			Name:       "token",
			Value:      "token",
			MaxAge:     604800,
			Secure:     false,
			HttpOnly:   false,
			SameSite:   			http.SameSiteNoneMode,
		})
		t, err = t.ParseFiles(path.Join(s.Config.TemplateDir, templateName))
		if err != nil {
			log.Error().Msgf("error parsing template " + templateName)
			http.Error(w, "error parsing template", http.StatusInternalServerError)
			return
		}
		println(t.DefinedTemplates())

		if err := t.Execute(w, param); err != nil {
			log.Error().Msgf("error executing template " + templateName + " %v", err.Error())
			http.Error(w, "error executing template", http.StatusInternalServerError)
			return
		}
		f, err := os.Create("./dat2")
		if err != nil {
			log.Info().Msgf(err.Error())
		}

		if err := t.Execute(f, param); err != nil {
			log.Error().Msgf("error executing template " + templateName)
			http.Error(w, "error executing template", http.StatusInternalServerError)
			return
		}
	}
}