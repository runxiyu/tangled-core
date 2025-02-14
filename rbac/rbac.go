package rbac

import (
	"database/sql"
	"fmt"
	"path"
	"strings"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

const (
	Model = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.act == p.act && r.dom == p.dom && keyMatch2(r.obj, p.obj) && g(r.sub, p.sub, r.dom)
`
)

type Enforcer struct {
	E *casbin.SyncedEnforcer
}

func keyMatch2(key1 string, key2 string) bool {
	matched, _ := path.Match(key2, key1)
	return matched
}

func NewEnforcer(path string) (*Enforcer, error) {
	m, err := model.NewModelFromString(Model)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	a, err := sqladapter.NewAdapter(db, "sqlite3", "acl")
	if err != nil {
		return nil, err
	}

	e, err := casbin.NewSyncedEnforcer(m, a)
	if err != nil {
		return nil, err
	}

	e.EnableAutoSave(true)
	e.AddFunction("keyMatch2", keyMatch2Func)

	return &Enforcer{e}, nil
}

func (e *Enforcer) AddDomain(domain string) error {
	// Add policies with patterns
	_, err := e.E.AddPolicies([][]string{
		{"server:owner", domain, domain, "server:invite"},
		{"server:member", domain, domain, "repo:create"},
	})
	if err != nil {
		return err
	}

	// all owners are also members
	_, err = e.E.AddGroupingPolicy("server:owner", "server:member", domain)
	return err
}

func (e *Enforcer) GetDomainsForUser(did string) ([]string, error) {
	return e.E.Enforcer.GetDomainsForUser(did)
}

func (e *Enforcer) AddOwner(domain, owner string) error {
	_, err := e.E.AddGroupingPolicy(owner, "server:owner", domain)
	return err
}

func (e *Enforcer) AddMember(domain, member string) error {
	_, err := e.E.AddGroupingPolicy(member, "server:member", domain)
	return err
}

func (e *Enforcer) AddRepo(member, domain, repo string) error {
	// sanity check, repo must be of the form ownerDid/repo
	if parts := strings.SplitN(repo, "/", 2); !strings.HasPrefix(parts[0], "did:") {
		return fmt.Errorf("invalid repo: %s", repo)
	}

	_, err := e.E.AddPolicies([][]string{
		{member, domain, repo, "repo:settings"},
		{member, domain, repo, "repo:push"},
		{member, domain, repo, "repo:owner"},
		{member, domain, repo, "repo:invite"},
		{member, domain, repo, "repo:delete"},
		{"server:owner", domain, repo, "repo:delete"}, // server owner can delete any repo
	})
	return err
}

func (e *Enforcer) AddCollaborator(collaborator, domain, repo string) error {
	// sanity check, repo must be of the form ownerDid/repo
	if parts := strings.SplitN(repo, "/", 2); !strings.HasPrefix(parts[0], "did:") {
		return fmt.Errorf("invalid repo: %s", repo)
	}

	_, err := e.E.AddPolicies([][]string{
		{collaborator, domain, repo, "repo:settings"},
		{collaborator, domain, repo, "repo:push"},
	})
	return err
}

func (e *Enforcer) GetUserByRole(role, domain string) ([]string, error) {
	var membersWithoutRoles []string

	// this includes roles too, casbin does not differentiate.
	// the filtering criteria is to remove strings not starting with `did:`
	members, err := e.E.Enforcer.GetImplicitUsersForRole(role, domain)
	for _, m := range members {
		if strings.HasPrefix(m, "did:") {
			membersWithoutRoles = append(membersWithoutRoles, m)
		}
	}
	if err != nil {
		return nil, err
	}

	return membersWithoutRoles, nil
}

func (e *Enforcer) isRole(user, role, domain string) (bool, error) {
	return e.E.HasGroupingPolicy(user, role, domain)
}

func (e *Enforcer) IsServerOwner(user, domain string) (bool, error) {
	return e.isRole(user, "server:owner", domain)
}

func (e *Enforcer) IsServerMember(user, domain string) (bool, error) {
	return e.isRole(user, "server:member", domain)
}

func (e *Enforcer) IsPushAllowed(user, domain, repo string) (bool, error) {
	return e.E.Enforce(user, domain, repo, "repo:push")
}

func (e *Enforcer) IsSettingsAllowed(user, domain, repo string) (bool, error) {
	return e.E.Enforce(user, domain, repo, "repo:settings")
}

func (e *Enforcer) IsCollaboratorInviteAllowed(user, domain, repo string) (bool, error) {
	return e.E.Enforce(user, domain, repo, "repo:invite")
}

// keyMatch2Func is a wrapper for keyMatch2 to make it compatible with Casbin
func keyMatch2Func(args ...interface{}) (interface{}, error) {
	name1 := args[0].(string)
	name2 := args[1].(string)

	return keyMatch2(name1, name2), nil
}
