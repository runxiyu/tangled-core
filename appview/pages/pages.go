package pages

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/sotangled/tangled/appview/auth"
	"github.com/sotangled/tangled/appview/db"
	"github.com/sotangled/tangled/types"
)

//go:embed templates/* static/*
var files embed.FS

type Pages struct {
	t map[string]*template.Template
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"split": func(s string) []string {
			return strings.Split(s, "\n")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"cond": func(cond interface{}, a, b string) string {
			if cond == nil {
				return b
			}

			if boolean, ok := cond.(bool); boolean && ok {
				return a
			}

			return b
		},
		"didOrHandle": func(did, handle string) string {
			if handle != "" {
				return fmt.Sprintf("@%s", handle)
			} else {
				return did
			}
		},
		"assoc": func(values ...string) ([][]string, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid assoc call, must have an even number of arguments")
			}
			pairs := make([][]string, 0)
			for i := 0; i < len(values); i += 2 {
				pairs = append(pairs, []string{values[i], values[i+1]})
			}
			return pairs, nil
		},
		"append": func(s []string, values ...string) []string {
			s = append(s, values...)
			return s
		},
		"timeFmt": humanize.Time,
		"length": func(v []string) int {
			return len(v)
		},
		"splitN": func(s, sep string, n int) []string {
			return strings.SplitN(s, sep, n)
		},
		"escapeHtml": func(s string) string {
			return template.HTMLEscapeString(s)
		},
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},
		"unwrapText": func(text string) string {
			paragraphs := strings.Split(text, "\n\n")

			for i, p := range paragraphs {
				lines := strings.Split(p, "\n")
				paragraphs[i] = strings.Join(lines, " ")
			}

			return strings.Join(paragraphs, "\n\n")
		},
	}
}

func NewPages() *Pages {
	templates := make(map[string]*template.Template)

	// Walk through embedded templates directory and parse all .html files
	err := fs.WalkDir(files, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			name := strings.TrimPrefix(path, "templates/")
			name = strings.TrimSuffix(name, ".html")

			if !strings.HasPrefix(path, "templates/layouts/") {
				// Add the page template on top of the base
				tmpl, err := template.New(name).
					Funcs(funcMap()).
					ParseFS(files, "templates/layouts/*.html", path)
				if err != nil {
					return fmt.Errorf("setting up template: %w", err)
				}

				templates[name] = tmpl
				log.Printf("loaded template: %s", name)
			}

			return nil
		}
		return nil
	})
	if err != nil {
		log.Fatalf("walking template dir: %v", err)
	}

	log.Printf("total templates loaded: %d", len(templates))

	return &Pages{
		t: templates,
	}
}

type LoginParams struct {
}

func (p *Pages) execute(name string, w io.Writer, params any) error {
	return p.t[name].ExecuteTemplate(w, "layouts/base", params)
}

func (p *Pages) executePlain(name string, w io.Writer, params any) error {
	return p.t[name].Execute(w, params)
}

func (p *Pages) executeRepo(name string, w io.Writer, params any) error {
	return p.t[name].ExecuteTemplate(w, "layouts/repobase", params)
}

func (p *Pages) Login(w io.Writer, params LoginParams) error {
	return p.executePlain("user/login", w, params)
}

type TimelineParams struct {
	LoggedInUser *auth.User
}

func (p *Pages) Timeline(w io.Writer, params TimelineParams) error {
	return p.execute("timeline", w, params)
}

type SettingsParams struct {
	LoggedInUser *auth.User
	PubKeys      []db.PublicKey
}

func (p *Pages) Settings(w io.Writer, params SettingsParams) error {
	return p.execute("settings/keys", w, params)
}

type KnotsParams struct {
	LoggedInUser  *auth.User
	Registrations []db.Registration
}

func (p *Pages) Knots(w io.Writer, params KnotsParams) error {
	return p.execute("knots", w, params)
}

type KnotParams struct {
	LoggedInUser *auth.User
	Registration *db.Registration
	Members      []string
	IsOwner      bool
}

func (p *Pages) Knot(w io.Writer, params KnotParams) error {
	return p.execute("knot", w, params)
}

type NewRepoParams struct {
	LoggedInUser *auth.User
	Knots        []string
}

func (p *Pages) NewRepo(w io.Writer, params NewRepoParams) error {
	return p.execute("repo/new", w, params)
}

type ProfilePageParams struct {
	LoggedInUser *auth.User
	UserDid      string
	UserHandle   string
	Repos        []db.Repo
}

func (p *Pages) ProfilePage(w io.Writer, params ProfilePageParams) error {
	return p.execute("user/profile", w, params)
}

type RepoInfo struct {
	Name            string
	OwnerDid        string
	OwnerHandle     string
	Description     string
	SettingsAllowed bool
}

func (r RepoInfo) OwnerWithAt() string {
	if r.OwnerHandle != "" {
		return fmt.Sprintf("@%s", r.OwnerHandle)
	} else {
		return r.OwnerDid
	}
}

func (r RepoInfo) FullName() string {
	return path.Join(r.OwnerWithAt(), r.Name)
}

func (r RepoInfo) GetTabs() [][]string {
	tabs := [][]string{
		{"overview", "/"},
		{"issues", "/issues"},
		{"pulls", "/pulls"},
	}

	if r.SettingsAllowed {
		tabs = append(tabs, []string{"settings", "/settings"})
	}

	return tabs
}

type RepoIndexParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	Active       string
	types.RepoIndexResponse
}

func (p *Pages) RepoIndexPage(w io.Writer, params RepoIndexParams) error {
	params.Active = "overview"
	return p.executeRepo("repo/index", w, params)
}

type RepoLogParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	types.RepoLogResponse
}

func (p *Pages) RepoLog(w io.Writer, params RepoLogParams) error {
	return p.execute("repo/log", w, params)
}

type RepoCommitParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	Active       string
	types.RepoCommitResponse
}

func (p *Pages) RepoCommit(w io.Writer, params RepoCommitParams) error {
	params.Active = "overview"
	return p.executeRepo("repo/commit", w, params)
}

type RepoTreeParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	Active       string
	BreadCrumbs  [][]string
	BaseTreeLink string
	BaseBlobLink string
	types.RepoTreeResponse
}

func (p *Pages) RepoTree(w io.Writer, params RepoTreeParams) error {
	params.Active = "overview"
	return p.execute("repo/tree", w, params)
}

type RepoBranchesParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	types.RepoBranchesResponse
}

func (p *Pages) RepoBranches(w io.Writer, params RepoBranchesParams) error {
	return p.executeRepo("repo/branches", w, params)
}

type RepoTagsParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	types.RepoTagsResponse
}

func (p *Pages) RepoTags(w io.Writer, params RepoTagsParams) error {
	return p.executeRepo("repo/tags", w, params)
}

type RepoBlobParams struct {
	LoggedInUser *auth.User
	RepoInfo     RepoInfo
	Active       string
	BreadCrumbs  [][]string
	types.RepoBlobResponse
}

func (p *Pages) RepoBlob(w io.Writer, params RepoBlobParams) error {
	params.Active = "overview"
	return p.executeRepo("repo/blob", w, params)
}

type RepoSettingsParams struct {
	LoggedInUser                *auth.User
	RepoInfo                    RepoInfo
	Collaborators               [][]string
	Active                      string
	IsCollaboratorInviteAllowed bool
}

func (p *Pages) RepoSettings(w io.Writer, params RepoSettingsParams) error {
	params.Active = "settings"
	return p.executeRepo("repo/settings", w, params)
}

func (p *Pages) Static() http.Handler {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		log.Fatalf("no static dir found? that's crazy: %v", err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}

func (p *Pages) Error500(w io.Writer) error {
	return p.execute("errors/500", w, nil)
}

func (p *Pages) Error404(w io.Writer) error {
	return p.execute("errors/404", w, nil)
}

func (p *Pages) Error503(w io.Writer) error {
	return p.execute("errors/503", w, nil)
}
