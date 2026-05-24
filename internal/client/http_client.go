package client

import (
	"bytes"
	"did_helper/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient HTTP client wrapper
type HTTPClient struct {
	client  *http.Client
	headers map[string]string
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second, // Default timeout 30 seconds
		},
		headers: make(map[string]string),
	}
}

// SetHeader sets a request header
func (c *HTTPClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders sets multiple request headers
func (c *HTTPClient) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		c.headers[k] = v
	}
}

// Get sends a GET request
func (c *HTTPClient) Get(url string) (int, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(body), nil
}

// Post sends a POST request
func (c *HTTPClient) Post(url string, body interface{}) (int, string, error) {
	var reader io.Reader

	// If body is not nil, serialize to JSON
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// Put sends a PUT request
func (c *HTTPClient) Put(url string, body interface{}) (int, string, error) {
	var reader io.Reader

	// If body is not nil, serialize to JSON
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// Delete sends a DELETE request
func (c *HTTPClient) Delete(url string) (int, string, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(body), nil
}

// addHeaders adds headers to request
func (c *HTTPClient) addHeaders(req *http.Request) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
}

func (c *HTTPClient) VerifyResponse(statusCode int, res string, err error) error {
	if err != nil && (statusCode != 201 || statusCode != 200) {
		return fmt.Errorf("failed to register DID: %w", err)

	}
	if statusCode != 201 && statusCode != 200 {
		if res == "" {
			return fmt.Errorf("failed to register DID: %w", err)
		} else {
			errResp := &models.ErrorResponse{}
			if err := json.Unmarshal([]byte(res), errResp); err != nil {
				return fmt.Errorf("failed to parse error response: %w", err)
			}
			if !errResp.Success {
				return fmt.Errorf("\n\rMessage:%s\r\nReason:%s\r\n", errResp.Message, errResp.Error)

			}
		}
	}
	return nil
}

// GetWithTicket sends a GET request with ticket header
func (c *HTTPClient) GetWithTicket(url string, ticket string) (int, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("ticket", ticket)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// PostWithTicket sends a POST request with ticket header
func (c *HTTPClient) PostWithTicket(url string, body interface{}, ticket string) (int, string, error) {
	var reader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ticket", ticket)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// PutWithTicket sends a PUT request with ticket header
func (c *HTTPClient) PutWithTicket(url string, body interface{}, ticket string) (int, string, error) {
	var reader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ticket", ticket)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// DeleteWithTicket sends a DELETE request with ticket header
func (c *HTTPClient) DeleteWithTicket(url string, ticket string) (int, string, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("ticket", ticket)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// GetWithTicketAndDID sends a GET request with ticket and DID headers
func (c *HTTPClient) GetWithTicketAndDID(url string, ticket string, did string) (int, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("ticket", ticket)
	req.Header.Set("X-DID", did)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// PostWithTicketAndDID sends a POST request with ticket and DID headers
func (c *HTTPClient) PostWithTicketAndDID(url string, body interface{}, ticket string, did string) (int, string, error) {
	var reader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ticket", ticket)
	req.Header.Set("X-DID", did)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// PutWithTicketAndDID sends a PUT request with ticket and DID headers
func (c *HTTPClient) PutWithTicketAndDID(url string, body interface{}, ticket string, did string) (int, string, error) {
	var reader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("Failed to serialize JSON: %w", err)
		}
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ticket", ticket)
	req.Header.Set("X-DID", did)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

// DeleteWithTicketAndDID sends a DELETE request with ticket and DID headers
func (c *HTTPClient) DeleteWithTicketAndDID(url string, ticket string, did string) (int, string, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("ticket", ticket)
	req.Header.Set("X-DID", did)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("Failed to read response: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}
