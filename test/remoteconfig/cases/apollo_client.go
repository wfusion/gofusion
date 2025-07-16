package cases

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gobwas/glob"

	"github.com/wfusion/gofusion/common/utils"
)

type apolloAdminClient struct {
	portalURL string
	appID     string
	env       string
	cluster   string
	token     string
	user      string
	client    *http.Client
}

func newApolloAdminClient(portalURL string) *apolloAdminClient {
	return &apolloAdminClient{
		portalURL: portalURL,
		appID:     "gofusion",
		cluster:   "default",
		token:     "e16e6897b788a18357a79a834f37e492f155c879",
		user:      "apollo",
		env:       "DEV",
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *apolloAdminClient) CreateNamespace(name, format string, isPublic bool, comment string) error {
	// /openapi/v1/apps/{appId}/appnamespaces
	url := fmt.Sprintf("%s/openapi/v1/apps/%s/appnamespaces", c.portalURL, c.appID)

	payload, _ := json.Marshal(map[string]interface{}{
		"name":                strings.TrimSuffix(name, fmt.Sprintf(".%s", format)),
		"appId":               c.appID,
		"format":              format, // properties、xml、json、yml、yaml
		"isPublic":            isPublic,
		"comment":             comment,
		"dataChangeCreatedBy": c.user,
	})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	return c.doRequest(req, utils.AnyPtr("Private AppNamespace*already exists!"), http.StatusOK)
}

func (c *apolloAdminClient) UpsertItem(namespace, key, value string) error {
	// /openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/items/{key}
	url := fmt.Sprintf("%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/items/%s?createIfNotExists=true",
		c.portalURL, c.env, c.appID, c.cluster, namespace, key)

	payload, _ := json.Marshal(map[string]string{
		"key":                      key,
		"value":                    value,
		"dataChangeCreatedBy":      c.user,
		"dataChangeLastModifiedBy": c.user,
	})

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	return c.doRequest(req, nil, http.StatusOK)
}

func (c *apolloAdminClient) DeleteItem(namespace, key string) error {
	// /openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/items/{key}?operator={operator}
	url := fmt.Sprintf("%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/items/%s?operator=%s",
		c.portalURL, c.env, c.appID, c.cluster, namespace, key, c.user)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil, http.StatusOK)
}

func (c *apolloAdminClient) PublishRelease(namespace string) error {
	// /openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/releases
	url := fmt.Sprintf("%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/releases",
		c.portalURL, c.env, c.appID, c.cluster, namespace)

	payload, _ := json.Marshal(map[string]string{
		"releaseTitle":   fmt.Sprintf("%s-release", time.Now().Format("20060102-150405")),
		"releaseComment": "Auto release for unit test",
		"releasedBy":     c.user,
	})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	return c.doRequest(req, nil, http.StatusOK)
}

type apolloRsp struct {
	Exception string      `json:"exception"`
	Message   string      `json:"message"`
	Status    json.Number `json:"status"`
	Timestamp string      `json:"timestamp"`
}

func (c *apolloAdminClient) doRequest(req *http.Request, ignoredBody *string, expectedStatusCodes ...int) (err error) {
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if len(expectedStatusCodes) == 0 {
		expectedStatusCodes = []int{http.StatusOK}
	}

	for _, code := range expectedStatusCodes {
		if resp.StatusCode == code {
			return
		}
	}

	body, _ := io.ReadAll(resp.Body)
	rsp := utils.MustJsonUnmarshal[apolloRsp](body)
	if ignoredBody != nil {
		if m, _ := glob.Compile(*ignoredBody); m.Match(rsp.Message) {
			return
		}
	}

	return fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
}
