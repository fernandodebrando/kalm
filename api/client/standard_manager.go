package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/kalmhq/kalm/api/log"
	"github.com/kalmhq/kalm/api/rbac"
	"github.com/kalmhq/kalm/api/resources"
	"github.com/kalmhq/kalm/controller/api/v1alpha1"
	"github.com/kalmhq/kalm/controller/controllers"
	"github.com/labstack/echo/v4"
	"html/template"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"strings"
	"sync"
	"time"
)

type StandardClientManager struct {
	*BaseClientManager

	PolicyAdapter *rbac.StringPolicyAdapter

	ClusterConfig *rest.Config

	// Access tokens, roleBindings, applications are rarely changed.
	// It is efficient to hold all roles and access tokens in memory to authorize requests.
	mut           *sync.RWMutex
	Applications  map[string]*coreV1.Namespace
	AccessTokens  map[string]*v1alpha1.AccessToken
	RoleBindings  map[string]*v1alpha1.RoleBinding
	StopWatchChan chan struct{}
}

func BuildClusterRolePolicies() string {
	return `
# cluster role policies
p, role_clusterViewer, view, *, *
p, role_clusterEditor, edit, *, *
p, role_clusterOwner, manage, *, *
g, role_clusterEditor, role_clusterViewer
g, role_clusterOwner, role_clusterEditor
`
}

func BuildRolePoliciesForNamespace(name string) string {
	t := template.Must(template.New("policy").Parse(`
# {{ .name }} application role policies
p, role_{{ .name }}Viewer, view, {{ .name }}, *
p, role_{{ .name }}Viewer, view, *, storageClasses
p, role_{{ .name }}Editor, edit, {{ .name }}, *
p, role_{{ .name }}Editor, view, *, registries
p, role_{{ .name }}Owner, manage, {{ .name }}, *
g, role_{{ .name }}Editor, role_{{ .name }}Viewer
g, role_{{ .name }}Owner, role_{{ .name }}Editor
`))

	strBuffer := &strings.Builder{}
	_ = t.Execute(strBuffer, map[string]string{"name": name})

	return strBuffer.String()
}

func GetPoliciesFromAccessToken(accessToken *resources.AccessToken) [][]string {
	var res = [][]string{}
	for _, rule := range accessToken.Rules {

		var obj string

		if rule.Namespace == "*" && rule.Name == "*" {
			obj = "*"
		} else {
			obj = fmt.Sprintf("%s/%s", rule.Kind, rule.Name)
		}

		res = append(res, []string{
			ToSafeSubject(accessToken.Name, v1alpha1.SubjectTypeUser),
			string(rule.Verb),
			rule.Namespace,
			obj,
		})
	}

	return res
}

func roleValueToPolicyValue(role, ns string) string {
	switch role {
	case v1alpha1.ClusterRoleViewer, v1alpha1.ClusterRoleEditor, v1alpha1.ClusterRoleOwner:
		return "role_" + role
	default:
		return fmt.Sprintf("role_%s%s", ns, strings.ToUpper(role[:1])+role[1:])
	}

}

func (m *StandardClientManager) UpdatePolicies() {
	var sb strings.Builder

	sb.WriteString(BuildClusterRolePolicies())

	for _, application := range m.Applications {
		if application.Labels == nil || application.Labels[controllers.KalmEnableLabelName] != controllers.KalmEnableLabelValue {
			continue
		}

		sb.WriteString(BuildRolePoliciesForNamespace(application.Name))
	}

	for _, accessToken := range m.AccessTokens {
		if accessToken.Spec.ExpiredAt != nil && accessToken.Spec.ExpiredAt.Time.Before(time.Now()) {
			continue
		}

		sb.WriteString(fmt.Sprintf("# policies for access token %s\n", accessToken.Name))

		for _, policy := range GetPoliciesFromAccessToken(&resources.AccessToken{Name: accessToken.Name, AccessTokenSpec: &accessToken.Spec}) {
			sb.WriteString(
				fmt.Sprintf(
					"p, %s, %s, %s, %s\n",
					policy[0],
					policy[1],
					policy[2],
					policy[3],
				),
			)
		}
	}

	for _, roleBinding := range m.RoleBindings {
		if roleBinding.Spec.ExpiredAt != nil && roleBinding.Spec.ExpiredAt.Time.Before(time.Now()) {
			continue
		}

		sb.WriteString(fmt.Sprintf("# policies for rolebinding %s\n", roleBinding.Name))
		sb.WriteString(fmt.Sprintf(
			"g, %s, %s\n",
			ToSafeSubject(roleBinding.Spec.Subject, roleBinding.Spec.SubjectType),
			roleValueToPolicyValue(roleBinding.Spec.Role, roleBinding.Namespace)),
		)
	}

	m.PolicyAdapter.SetPoliciesString(sb.String())

	if err := m.RBACEnforcer.LoadPolicy(); err != nil {
		// the policy is important. Stale policies can be harmful.
		panic(err)
	}
}

func (m *StandardClientManager) GetDefaultClusterConfig() *rest.Config {
	return m.ClusterConfig
}

func (m *StandardClientManager) GetClientInfoFromToken(tokenString string) (*ClientInfo, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	accessToken, ok := m.AccessTokens[v1alpha1.GetAccessTokenNameFromToken(tokenString)]

	if !ok {
		return nil, errors.NewUnauthorized("access token not exist")
	}

	if accessToken.Spec.ExpiredAt != nil && accessToken.Spec.ExpiredAt.Time.Before(time.Now()) {
		return nil, errors.NewUnauthorized("access token is expired")
	}

	clientInfo := &ClientInfo{
		Cfg:           m.ClusterConfig,
		Name:          accessToken.Name,
		Email:         accessToken.Name,
		EmailVerified: false,
		Groups:        []string{},
	}

	return clientInfo, nil
}

func (m *StandardClientManager) SetImpersonation(clientInfo *ClientInfo, rawImpersonation string) {
	if rawImpersonation != "" && m.CanManageCluster(clientInfo) {
		impersonation, impersonationType, err := parseImpersonationString(rawImpersonation)

		if err == nil {
			clientInfo.Impersonation = impersonation
			clientInfo.ImpersonationType = impersonationType
		} else {
			log.Error(err, "parse impersonation raw string failed")
		}
	}
}

func (m *StandardClientManager) GetClientInfoFromContext(c echo.Context) (*ClientInfo, error) {
	// If the Authorization Header is not empty, use the bearer token as k8s token.
	if token := extractAuthTokenFromClientRequestContext(c); token != "" {
		clientInfo, err := m.GetClientInfoFromToken(token)
		if err != nil {
			return nil, err
		}

		m.SetImpersonation(clientInfo, c.Request().Header.Get("Kalm-Impersonation"))
		return clientInfo, nil
	}

	// And the kalm-sso-userinfo header is not empty.
	// This header will be removed at ingress route level. Only auth proxy can set this header, So it's safe to trust this value.
	if c.Request().Header.Get("Kalm-Sso-Userinfo") != "" {
		claimsBytes, err := base64.RawStdEncoding.DecodeString(c.Request().Header.Get("Kalm-Sso-Userinfo"))

		if err != nil {
			return nil, err
		}

		var clientInfo ClientInfo

		err = json.Unmarshal(claimsBytes, &clientInfo)

		if err != nil {
			return nil, err
		}

		clientInfo.Cfg = m.ClusterConfig
		clientInfo.Impersonation = ""

		m.SetImpersonation(&clientInfo, c.Request().Header.Get("Kalm-Impersonation"))
		return &clientInfo, nil
	}

	return nil, errors.NewUnauthorized("")
}

// Since the token is validated by api server, so we don't need to valid the token again here.
func tryToParseEntityFromToken(tokenString string) string {
	if tokenString == "" {
		return "unknown"
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})

	if err != nil {
		return "token"
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims["sub"].(string)
	}

	return "token"
}

func NewStandardClientManager(cfg *rest.Config) *StandardClientManager {
	policyAdapter := rbac.NewStringPolicyAdapter(``)

	manager := &StandardClientManager{
		BaseClientManager: NewBaseClientManager(policyAdapter),
		PolicyAdapter:     policyAdapter,
		ClusterConfig:     cfg,
		mut:               &sync.RWMutex{},
		Applications:      make(map[string]*coreV1.Namespace),
		AccessTokens:      make(map[string]*v1alpha1.AccessToken),
		RoleBindings:      make(map[string]*v1alpha1.RoleBinding),
		StopWatchChan:     make(chan struct{}),
	}

	go setupResourcesWatcher(cfg, manager)
	go policyRegenerateLoop(manager)

	return manager
}

// Run per minute to remove expired access tokens and role bindings
func policyRegenerateLoop(manager *StandardClientManager) {
	for {
		manager.mut.Lock()
		manager.UpdatePolicies()
		manager.mut.Unlock()
		time.Sleep(1 * time.Minute)
	}
}

func setupResourcesWatcher(cfg *rest.Config, manager *StandardClientManager) {
	informerCache, err := cache.New(cfg, cache.Options{})

	if err != nil {
		log.Error(err, "new cache error")
		panic(err)
	}

	if informer, err := informerCache.GetInformer(context.Background(), &coreV1.Namespace{}); err == nil {
		informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if namespace, ok := obj.(*coreV1.Namespace); ok {
					manager.Applications[namespace.Name] = namespace
					manager.UpdatePolicies()
				}
			},
			DeleteFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if namespace, ok := obj.(*coreV1.Namespace); ok {
					delete(manager.Applications, namespace.Name)
					manager.UpdatePolicies()
				}
			},
			UpdateFunc: func(oldObj, obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if namespace, ok := obj.(*coreV1.Namespace); ok {
					manager.Applications[namespace.Name] = namespace
					manager.UpdatePolicies()
				}
			},
		})
	} else {
		log.Error(err, "get informer error")
		panic(err)
	}

	if informer, err := informerCache.GetInformer(context.Background(), &v1alpha1.RoleBinding{}); err == nil {
		informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if roleBinding, ok := obj.(*v1alpha1.RoleBinding); ok {
					manager.RoleBindings[getNamespacedName(roleBinding.ObjectMeta)] = roleBinding
					manager.UpdatePolicies()
				}
			},
			DeleteFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if roleBinding, ok := obj.(*v1alpha1.RoleBinding); ok {
					delete(manager.RoleBindings, getNamespacedName(roleBinding.ObjectMeta))
					manager.UpdatePolicies()
				}
			},
			UpdateFunc: func(oldObj, obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if roleBinding, ok := obj.(*v1alpha1.RoleBinding); ok {
					manager.RoleBindings[getNamespacedName(roleBinding.ObjectMeta)] = roleBinding
					manager.UpdatePolicies()
				}
			},
		})
	} else {
		log.Error(err, "get informer error")
		panic(err)
	}

	if informer, err := informerCache.GetInformer(context.Background(), &v1alpha1.AccessToken{}); err == nil {
		informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if accessToken, ok := obj.(*v1alpha1.AccessToken); ok {
					manager.AccessTokens[accessToken.Name] = accessToken
					manager.UpdatePolicies()
				}
			},
			DeleteFunc: func(obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if accessToken, ok := obj.(*v1alpha1.AccessToken); ok {
					delete(manager.AccessTokens, accessToken.Name)
					manager.UpdatePolicies()
				}
			},
			UpdateFunc: func(oldObj, obj interface{}) {
				manager.mut.Lock()
				defer manager.mut.Unlock()
				if accessToken, ok := obj.(*v1alpha1.AccessToken); ok {
					manager.AccessTokens[accessToken.Name] = accessToken
					manager.UpdatePolicies()
				}
			},
		})
	} else {
		log.Error(err, "get informer error")
		panic(err)
	}

	informerCache.Start(manager.StopWatchChan)
}

func getNamespacedName(metaObj metaV1.ObjectMeta) string {
	return fmt.Sprintf("%s-%s", metaObj.Namespace, metaObj.Name)
}
