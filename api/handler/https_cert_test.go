package handler

import (
	"fmt"
	client2 "github.com/kalmhq/kalm/api/client"
	"github.com/kalmhq/kalm/api/resources"
	"github.com/kalmhq/kalm/controller/api/v1alpha1"
	"github.com/kalmhq/kalm/controller/controllers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"testing"
)

type HttpsCertTestSuite struct {
	WithControllerTestSuite
}

func TestHttpsCertTestSuite(t *testing.T) {
	suite.Run(t, new(HttpsCertTestSuite))
}

func (suite *HttpsCertTestSuite) SetupSuite() {
	suite.WithControllerTestSuite.SetupSuite()

	suite.ensureNamespaceExist("istio-system")
	suite.ensureNamespaceExist(controllers.CertManagerNamespace)
}

func (suite *HttpsCertTestSuite) TearDownTest() {
	suite.ensureObjectDeleted(&v1alpha1.HttpsCert{ObjectMeta: metaV1.ObjectMeta{Name: "foobar-cert"}})
	suite.ensureObjectDeleted(&coreV1.Secret{ObjectMeta: metaV1.ObjectMeta{Name: "kalm-self-managed-foobar-cert", Namespace: "istio-system"}})
}

func (suite *HttpsCertTestSuite) TestGetEmptyHttpsCertList() {
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterViewerRole(),
		},
		Method: http.MethodGet,
		Path:   "/v1alpha1/httpscerts",
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "viewer", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var res []resources.HttpsCert
			rec.BodyAsJSON(&res)

			suite.Equal(200, rec.Code)
		},
	})
}

func (suite *HttpsCertTestSuite) TestCreateHttpsCert() {
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts",
		Body: `{
  "name":    "foobar-cert",
  "httpsCertIssuer":  "foobar-issuer",
  "domains": ["example.com"]
}`,
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var httpsCert resources.HttpsCert
			var res v1alpha1.HttpsCertList

			rec.BodyAsJSON(&httpsCert)

			suite.Equal(201, rec.Code)
			suite.Equal("foobar-cert", httpsCert.Name)

			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			suite.Equal("foobar-cert", res.Items[0].Name)
			//fmt.Println("item:", res.Items[0])
			suite.Equal("foobar-issuer", res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("example.com", strings.Join(res.Items[0].Spec.Domains, ""))
		},
	})

	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterViewerRole(),
		},
		Method: http.MethodGet,
		Path:   "/v1alpha1/httpscerts",
		Body: `{
  "name":    "foobar-cert",
  "httpsCertIssuer":  "foobar-issuer",
  "domains": ["example.com"]
}`,
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "viewer", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var resList []resources.HttpsCertResp
			rec.BodyAsJSON(&resList)

			suite.Equal(200, rec.Code)
			suite.Equal(1, len(resList))
			suite.Equal(string(coreV1.ConditionUnknown), resList[0].Ready)
			suite.Equal(resources.ReasonForNoReadyConditions, resList[0].Reason)
		},
	})

}

func (suite *HttpsCertTestSuite) TestCreateHttpsCertWithoutName() {
	var httpsCert resources.HttpsCert

	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts",
		Body: `{
  "httpsCertIssuer":  "foobar-issuer",
  "domains": ["example.com"]
}`,
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var res v1alpha1.HttpsCertList
			rec.BodyAsJSON(&httpsCert)

			suite.Equal(201, rec.Code)
			suite.NotEqual("", httpsCert.Name)

			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			fmt.Println(res.Items[0].Name)
			suite.True(strings.HasPrefix(res.Items[0].Name, "example-com-"))
			suite.Equal("foobar-issuer", res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("example.com", strings.Join(res.Items[0].Spec.Domains, ""))

		},
		CleanUp: func(rc *TestRequestContext, s *echo.Echo) {
			rec := BaseRequest(s, http.MethodDelete, "/v1alpha1/httpscerts/"+httpsCert.Name, nil, map[string]string{
				echo.HeaderAuthorization: "Bearer " + client2.ToFakeToken(rc.User),
			})

			suite.Equal(200, rec.Code)
		},
	})
}

func (suite *HttpsCertTestSuite) TestCreateHttpsCertWithoutSetIssuer() {
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts",
		Body: `{
  "name":    "foobar-cert",
  "domains": ["example.com"]
}`,
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var httpsCert resources.HttpsCert
			var res v1alpha1.HttpsCertList

			rec.BodyAsJSON(&httpsCert)

			suite.Equal(201, rec.Code)
			suite.Equal("foobar-cert", httpsCert.Name)

			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			suite.Equal("foobar-cert", res.Items[0].Name)
			suite.Equal(v1alpha1.DefaultHTTP01IssuerName, res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("example.com", strings.Join(res.Items[0].Spec.Domains, ""))
		},
	})
}

const tlsCert = `-----BEGIN CERTIFICATE-----
MIIFVjCCBD6gAwIBAgISBPNCxpUJsb9iD+AX7DviehGrMA0GCSqGSIb3DQEBCwUA
MEoxCzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MSMwIQYDVQQD
ExpMZXQncyBFbmNyeXB0IEF1dGhvcml0eSBYMzAeFw0yMDA1MjAwMzI5MzNaFw0y
MDA4MTgwMzI5MzNaMBoxGDAWBgNVBAMTD2hlbGxvLmthcHAubGl2ZTCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBAJ48RtSGIUl66BXE5H7TF81dm2JHWxS9
WaPLB9fw+7aE7Q80MqNemxC9919eiLsY43/5vE+oGyqCxy5OA+NjNhqkRyfRtLOy
C+qT30s+bSGVc7iwRqyBSA/1tVjvlio+bD3jmiKP8G4fX0MswJmUIhUqDjrgcz73
7WCn0SxfJrxRVihgQ0BYdwn7rhXd9owQK5KIYG80a/oLwnsplJCzI3MeDzhLz/oK
pcaPI8qoLH4Bxb7Od/tKODpp80ub6c4x+M+qI62Goo50+Sm6vwVzc8CsSlz2lGDN
608tBWDZ6HJrn0ogBa/qUFdSFXrQcpeVNVi+/suT/+wGJ1KtiDInM0ECAwEAAaOC
AmQwggJgMA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYB
BQUHAwIwDAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQU88bxblZUQdX7RYMsUnKxUTwK
Z04wHwYDVR0jBBgwFoAUqEpqYwR93brm0Tm3pkVl7/Oo7KEwbwYIKwYBBQUHAQEE
YzBhMC4GCCsGAQUFBzABhiJodHRwOi8vb2NzcC5pbnQteDMubGV0c2VuY3J5cHQu
b3JnMC8GCCsGAQUFBzAChiNodHRwOi8vY2VydC5pbnQteDMubGV0c2VuY3J5cHQu
b3JnLzAaBgNVHREEEzARgg9oZWxsby5rYXBwLmxpdmUwTAYDVR0gBEUwQzAIBgZn
gQwBAgEwNwYLKwYBBAGC3xMBAQEwKDAmBggrBgEFBQcCARYaaHR0cDovL2Nwcy5s
ZXRzZW5jcnlwdC5vcmcwggEEBgorBgEEAdZ5AgQCBIH1BIHyAPAAdgDwlaRZ8gDR
gkAQLS+TiI6tS/4dR+OZ4dA0prCoqo6ycwAAAXIwWAAmAAAEAwBHMEUCIQD/weuk
7dWqw7iswofV7vt4ANxIvVFKfynHik1tDWGX2QIgUZwvdLxNjduE15kPB5G3zpbp
7I8Y2VIWIgxyZIGR3BEAdgCyHgXMi6LNiiBOh2b5K7mKJSBna9r6cOeySVMt74uQ
XgAAAXIwWAAVAAAEAwBHMEUCIGJwq3BvFcWn8CwRwXsMkOR2FUKAV1XcDwcJsbJa
jFsgAiEA5dqDJ0oL2V2ItThyNQRGTD7WvVKjghCL+EIO5yaYZaswDQYJKoZIhvcN
AQELBQADggEBAJJ8mKQ+IyuFlOMijD5RhU3U7l8rR5R/f9ITRUK5Q3NgkmhvNG8t
wBCcnVr3nNnKFYloLJ0rBSPyqs/nE9KljHzhZoootkP8PfXHe7A6FOR8xohzqHR0
u54xd1p+jTluVOE+Ofwa32VZ4VkIUIezyoSpLz1xk00tVtFlIrBn1Bk2vskTA5XK
znkm2KnVBuj75tteXjjMthi+yKW1Bfd1I2mCuSz8sylKyXx+2sX6YVR5o1+NamBi
7mK92Uhdb4Zq21RpDNYWrITAjVIunofNSgGjFu165ZGvPCMG0DwhvFnzWb97dsB5
fZLKGiQmUq6JTawO7JIZDdVDK7zZQTsEQG8=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIEkjCCA3qgAwIBAgIQCgFBQgAAAVOFc2oLheynCDANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTE2MDMxNzE2NDA0NloXDTIxMDMxNzE2NDA0Nlow
SjELMAkGA1UEBhMCVVMxFjAUBgNVBAoTDUxldCdzIEVuY3J5cHQxIzAhBgNVBAMT
GkxldCdzIEVuY3J5cHQgQXV0aG9yaXR5IFgzMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEAnNMM8FrlLke3cl03g7NoYzDq1zUmGSXhvb418XCSL7e4S0EF
q6meNQhY7LEqxGiHC6PjdeTm86dicbp5gWAf15Gan/PQeGdxyGkOlZHP/uaZ6WA8
SMx+yk13EiSdRxta67nsHjcAHJyse6cF6s5K671B5TaYucv9bTyWaN8jKkKQDIZ0
Z8h/pZq4UmEUEz9l6YKHy9v6Dlb2honzhT+Xhq+w3Brvaw2VFn3EK6BlspkENnWA
a6xK8xuQSXgvopZPKiAlKQTGdMDQMc2PMTiVFrqoM7hD8bEfwzB/onkxEz0tNvjj
/PIzark5McWvxI0NHWQWM6r6hCm21AvA2H3DkwIDAQABo4IBfTCCAXkwEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwfwYIKwYBBQUHAQEEczBxMDIG
CCsGAQUFBzABhiZodHRwOi8vaXNyZy50cnVzdGlkLm9jc3AuaWRlbnRydXN0LmNv
bTA7BggrBgEFBQcwAoYvaHR0cDovL2FwcHMuaWRlbnRydXN0LmNvbS9yb290cy9k
c3Ryb290Y2F4My5wN2MwHwYDVR0jBBgwFoAUxKexpHsscfrb4UuQdf/EFWCFiRAw
VAYDVR0gBE0wSzAIBgZngQwBAgEwPwYLKwYBBAGC3xMBAQEwMDAuBggrBgEFBQcC
ARYiaHR0cDovL2Nwcy5yb290LXgxLmxldHNlbmNyeXB0Lm9yZzA8BgNVHR8ENTAz
MDGgL6AthitodHRwOi8vY3JsLmlkZW50cnVzdC5jb20vRFNUUk9PVENBWDNDUkwu
Y3JsMB0GA1UdDgQWBBSoSmpjBH3duubRObemRWXv86jsoTANBgkqhkiG9w0BAQsF
AAOCAQEA3TPXEfNjWDjdGBX7CVW+dla5cEilaUcne8IkCJLxWh9KEik3JHRRHGJo
uM2VcGfl96S8TihRzZvoroed6ti6WqEBmtzw3Wodatg+VyOeph4EYpr/1wXKtx8/
wApIvJSwtmVi4MFU5aMqrSDE6ea73Mj2tcMyo5jMd6jmeWUHK8so/joWUoHOUgwu
X4Po1QYz+3dszkDqMp4fklxBwXRsW10KXzPMTZ+sOPAveyxindmjkW8lGy+QsRlG
PfZ+G6Z6h7mjem0Y+iWlkYcV4PIWL1iwBi8saCbGS5jN2p8M+X+Q7UNKEkROb3N6
KOqkqm57TH2H3eDJAkSnh6/DNFu0Qg==
-----END CERTIFICATE-----`

func (suite *HttpsCertTestSuite) TestUploadHttpsCert() {
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts/upload",
		Body: map[string]interface{}{
			"name":                      "foobar-cert",
			"isSelfManaged":             true,
			"selfManagedCertContent":    tlsCert,
			"selfManagedCertPrivateKey": "",
		},
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var httpsCert resources.HttpsCert
			rec.BodyAsJSON(&httpsCert)

			suite.Equal(201, rec.Code)
			suite.Equal("foobar-cert", httpsCert.Name)

			var res v1alpha1.HttpsCertList
			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			suite.Equal("foobar-cert", res.Items[0].Name)
			//fmt.Println("item:", res.Items[0])
			suite.Equal(true, res.Items[0].Spec.IsSelfManaged)
			suite.Equal("", res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("hello.kapp.live", strings.Join(res.Items[0].Spec.Domains, ""))
			suite.Equal("kalm-self-managed-foobar-cert", res.Items[0].Spec.SelfManagedCertSecretName)

			// sec
			var sec coreV1.Secret
			err = suite.Get("istio-system", "kalm-self-managed-foobar-cert", &sec)
			suite.Nil(err)
			suite.Equal(sec.Data["tls.key"], []byte(""))
			suite.Equal(sec.Data["tls.crt"], []byte(tlsCert))
		},
	})

	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterViewerRole(),
		},
		Method: http.MethodGet,
		Path:   "/v1alpha1/httpscerts",
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "viewer", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var resList []resources.HttpsCert
			rec.BodyAsJSON(&resList)
			suite.Equal(200, rec.Code)
			suite.Equal(1, len(resList))
			suite.Equal("hello.kapp.live", strings.Join(resList[0].Domains, ""))
		},
	})
}

func (suite *HttpsCertTestSuite) TestUpdateSelfManagedHttpsCert() {
	// Upload
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts/upload",
		Body: map[string]interface{}{
			"name":                      "foobar-cert",
			"isSelfManaged":             true,
			"selfManagedCertContent":    tlsCert,
			"selfManagedCertPrivateKey": "",
		},
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			suite.Equal(201, rec.Code)
		},
	})

	// update
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPut,
		Path:   "/v1alpha1/httpscerts/foobar-cert",
		Body: map[string]interface{}{
			"name":                      "foobar-cert",
			"isSelfManaged":             true,
			"selfManagedCertContent":    tlsCert,
			"selfManagedCertPrivateKey": "updatedPrvKey",
		},
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			suite.Equal(200, rec.Code)
			var res v1alpha1.HttpsCertList
			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			suite.Equal("foobar-cert", res.Items[0].Name)
			suite.Equal(true, res.Items[0].Spec.IsSelfManaged)
			suite.Equal("", res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("hello.kapp.live", strings.Join(res.Items[0].Spec.Domains, ""))
			suite.Equal("kalm-self-managed-foobar-cert", res.Items[0].Spec.SelfManagedCertSecretName)

			// sec
			var sec coreV1.Secret
			err = suite.Get("istio-system", "kalm-self-managed-foobar-cert", &sec)
			suite.Nil(err)
			suite.Equal(sec.Data["tls.key"], []byte("updatedPrvKey"))
		},
	})
}

func (suite *HttpsCertTestSuite) TestDeleteHttpsCert() {
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodPost,
		Path:   "/v1alpha1/httpscerts",
		Body: `{
  "name":    "foobar-cert",
  "httpsCertIssuer":  "foobar-issuer",
  "domains": ["example.com"]
}`,
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			var httpsCert resources.HttpsCert
			rec.BodyAsJSON(&httpsCert)

			suite.Equal(201, rec.Code)
			suite.Equal("foobar-cert", httpsCert.Name)

			var res v1alpha1.HttpsCertList
			err := suite.List(&res)
			suite.Nil(err)

			suite.Equal(1, len(res.Items))
			suite.Equal("foobar-cert", res.Items[0].Name)
			//fmt.Println("item:", res.Items[0])
			suite.Equal("foobar-issuer", res.Items[0].Spec.HttpsCertIssuer)
			suite.Equal("example.com", strings.Join(res.Items[0].Spec.Domains, ""))
		},
	})

	// Delete
	suite.DoTestRequest(&TestRequestContext{
		Roles: []string{
			GetClusterEditorRole(),
		},
		Method: http.MethodDelete,
		Path:   "/v1alpha1/httpscerts/foobar-cert",
		TestWithoutRoles: func(rec *ResponseRecorder) {
			suite.IsMissingRoleError(rec, "editor", "cluster")
		},
		TestWithRoles: func(rec *ResponseRecorder) {
			suite.Equal(200, rec.Code)

			var res v1alpha1.HttpsCertList
			err := suite.List(&res)
			suite.Nil(err)
			suite.Equal(0, len(res.Items))
		},
	})
}
