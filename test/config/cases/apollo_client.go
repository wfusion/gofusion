package cases

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type ApolloAdminClient struct {
	portalURL string
	token     string
	user      string
	client    *http.Client
}

func NewApolloAdminClient(portalURL string) *ApolloAdminClient {
	return &ApolloAdminClient{
		portalURL: portalURL,
		token:     "e16e6897b788a18357a79a834f37e492f155c879",
		user:      "apollo",
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ApolloAdminClient) CreateOrUpdateItem(appID, cluster, namespace, key, value string) error {
	url := fmt.Sprintf("%s/apps/%s/clusters/%s/namespaces/%s/items/%s", c.portalURL, appID, cluster, namespace, key)

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
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create/update item, status: %s", resp.Status)
	}
	return nil
}

func (c *ApolloAdminClient) DeleteItem(appID, cluster, namespace, key string) error {
	url := fmt.Sprintf("%s/apps/%s/clusters/%s/namespaces/%s/items/%s?operator=%s", c.portalURL, appID, cluster, namespace, key, c.user)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete item, status: %s", resp.Status)
	}
	return nil
}

func (c *ApolloAdminClient) PublishRelease(appID, cluster, namespace string) error {
	url := fmt.Sprintf("%s/apps/%s/clusters/%s/namespaces/%s/releases", c.portalURL, appID, cluster, namespace)

	payload, _ := json.Marshal(map[string]string{
		"releaseName":    fmt.Sprintf("test-release-%d", time.Now().Unix()),
		"releaseComment": "Auto release for unit test",
		"releasedBy":     c.user,
	})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to publish release, status: %s", resp.Status)
	}
	return nil
}
