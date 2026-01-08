package contact

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitre/gocat/output"
)

var (
	apiBeacon = "/beacon"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"
	// Retry configuration
	maxRetries = 3
	initialRetryDelay = time.Second * 2
	maxRetryDelay = time.Second * 30
	httpTimeout = time.Second * 30
)

//API communicates through HTTP
type API struct {
	name string
	client *http.Client
	upstreamDestAddr string
}

func init() {
	CommunicationChannels["HTTP"] = &API{ name: "HTTP" }
}

//GetInstructions sends a beacon and returns response.
func (a *API) GetBeaconBytes(profile map[string]interface{}) []byte {
	data, err := json.Marshal(profile)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot request beacon. Error with profile marshal: %s", err.Error()))
		return nil
	} else {
		address := fmt.Sprintf("%s%s", a.upstreamDestAddr, apiBeacon)
		return a.request(address, data)
	}
}

// Return the file bytes for the requested payload.
func (a *API) GetPayloadBytes(profile map[string]interface{}, payload string) ([]byte, string) {
    var payloadBytes []byte
    var filename string
    platform := profile["platform"]
    if platform != nil {
		address := fmt.Sprintf("%s/file/download", a.upstreamDestAddr)
		
		for attempt := 0; attempt <= maxRetries; attempt++ {
			req, err := http.NewRequest("POST", address, nil)
			if err != nil {
				output.VerbosePrint(fmt.Sprintf("[-] Failed to create HTTP request: %s", err.Error()))
				return nil, ""
			}
			req.Header.Set("file", payload)
			req.Header.Set("platform", platform.(string))
			req.Header.Set("paw", profile["paw"].(string))
			
			resp, err := a.client.Do(req)
			if err != nil {
				if attempt < maxRetries && isRetryableError(err) {
					delay := calculateRetryDelay(attempt)
					output.VerbosePrint(fmt.Sprintf("[!] Payload request failed (attempt %d/%d): %s. Retrying in %v", 
						attempt+1, maxRetries+1, err.Error(), delay))
					time.Sleep(delay)
					continue
				}
				output.VerbosePrint(fmt.Sprintf("[-] Error sending payload request after %d attempts: %s", attempt+1, err.Error()))
				return nil, ""
			}
			
			if resp.StatusCode == ok {
				buf, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					if attempt < maxRetries {
						delay := calculateRetryDelay(attempt)
						output.VerbosePrint(fmt.Sprintf("[!] Error reading payload response (attempt %d/%d): %s. Retrying in %v", 
							attempt+1, maxRetries+1, err.Error(), delay))
						time.Sleep(delay)
						continue
					}
					output.VerbosePrint(fmt.Sprintf("[-] Error reading HTTP response after %d attempts: %s", attempt+1, err.Error()))
					return nil, ""
				}
				
				payloadBytes = buf
				if name_header, ok := resp.Header["Filename"]; ok {
					filename = filepath.Join(name_header[0])
				} else {
					output.VerbosePrint("[-] HTTP response missing Filename header.")
				}
				
				// Success - log if this was a retry
				if attempt > 0 {
					output.VerbosePrint(fmt.Sprintf("[+] Payload request succeeded on attempt %d", attempt+1))
				}
				break
			} else {
				resp.Body.Close()
				if attempt < maxRetries {
					delay := calculateRetryDelay(attempt)
					output.VerbosePrint(fmt.Sprintf("[!] Payload request returned status %d (attempt %d/%d). Retrying in %v", 
						resp.StatusCode, attempt+1, maxRetries+1, delay))
					time.Sleep(delay)
					continue
				}
				output.VerbosePrint(fmt.Sprintf("[-] Payload request returned status %d after %d attempts", resp.StatusCode, attempt+1))
			}
		}
    }
	return payloadBytes, filename
}

//C2RequirementsMet determines if sandcat can use the selected comm channel
func (a *API) C2RequirementsMet(profile map[string]interface{}, c2Config map[string]string) (bool, map[string]string) {
	output.VerbosePrint(fmt.Sprintf("Beacon API=%s", apiBeacon))
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Handle proxy gateway configuration.
	if proxyUrlStr, ok := c2Config["httpProxyGateway"]; ok && len(proxyUrlStr) > 0 {
		proxyUrl, err := url.Parse(proxyUrlStr)
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[!] Error - could not establish HTTP proxy requirements: %s", err.Error()))
			return false, nil
		}
		http.DefaultTransport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
	}
	a.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   httpTimeout,
	}

	return true, nil
}

func (a *API) SetUpstreamDestAddr(upstreamDestAddr string) {
	a.upstreamDestAddr = upstreamDestAddr
}

// SendExecutionResults will send the execution results to the upstream destination.
func (a *API) SendExecutionResults(profile map[string]interface{}, result map[string]interface{}) {
	address := fmt.Sprintf("%s%s", a.upstreamDestAddr, apiBeacon)
	profileCopy := make(map[string]interface{})
	for k,v := range profile {
		profileCopy[k] = v
	}
	results := make([]map[string]interface{}, 1)
	results[0] = result
	profileCopy["results"] = results
	data, err := json.Marshal(profileCopy)
	if err != nil {
		output.VerbosePrint(fmt.Sprintf("[-] Cannot send results. Error with profile marshal: %s", err.Error()))
	} else {
		a.request(address, data)
	}
}

func (a *API) GetName() string {
	return a.name
}

func (a *API) UploadFileBytes(profile map[string]interface{}, uploadName string, data []byte) error {
	uploadUrl := a.upstreamDestAddr + "/file/upload"

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Set up the form
		requestBody := bytes.Buffer{}
		contentType, err := createUploadForm(&requestBody, data, uploadName)
		if err != nil {
			return err
		}

		// Set up the request
		headers := map[string]string{
			"Content-Type": contentType,
			"X-Request-Id": fmt.Sprintf("%s-%s", profile["host"].(string), profile["paw"].(string)),
			"User-Agent": userAgent,
			"X-Paw": profile["paw"].(string),
			"X-Host": profile["host"].(string),
		}
		req, err := createUploadRequest(uploadUrl, &requestBody, headers)
		if err != nil {
			return err
		}

		// Perform request and process response
		resp, err := a.client.Do(req)
		if err != nil {
			if attempt < maxRetries && isRetryableError(err) {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] File upload failed (attempt %d/%d): %s. Retrying in %v", 
					attempt+1, maxRetries+1, err.Error(), delay))
				time.Sleep(delay)
				continue
			}
			return err
		}
		
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// Success - log if this was a retry
			if attempt > 0 {
				output.VerbosePrint(fmt.Sprintf("[+] File upload succeeded on attempt %d", attempt+1))
			}
			return nil
		} else {
			if attempt < maxRetries {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] File upload returned status %d (attempt %d/%d). Retrying in %v", 
					resp.StatusCode, attempt+1, maxRetries+1, delay))
				time.Sleep(delay)
				continue
			}
			return errors.New(fmt.Sprintf("Non-successful HTTP response status code: %d after %d attempts", resp.StatusCode, attempt+1))
		}
	}
	return nil
}

func (a *API) SupportsContinuous() bool {
    return false
}

func createUploadForm(requestBody *bytes.Buffer, data []byte, uploadName string) (string, error) {
	writer := multipart.NewWriter(requestBody)
	defer writer.Close()
	dataReader := bytes.NewReader(data)
	formWriter, err := writer.CreateFormFile("file", uploadName)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(formWriter, dataReader); err != nil {
		return "", err
	}
	return writer.FormDataContentType(), nil
}

func createUploadRequest(uploadUrl string, requestBody *bytes.Buffer, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest("POST", uploadUrl, requestBody)
	if err != nil {
		return nil, err
	}
	for header, val := range headers {
		req.Header.Set(header, val)
	}
	return req, nil
}

// isRetryableStatusCode determines if an HTTP status code should trigger a retry
func isRetryableStatusCode(statusCode int) bool {
	// Server errors (5xx) and some client errors that might be temporary
	return statusCode >= 500 || statusCode == 408 || statusCode == 429
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Network connectivity issues that are often temporary
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"timeout",
		"temporary failure",
		"wsarecv",
		"wsasend",
		"no such host", // DNS issues
		"network is unreachable",
		"broken pipe",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}
	return false
}

// calculateRetryDelay calculates exponential backoff delay with jitter
func calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(1<<uint(attempt)) * initialRetryDelay
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	// Add jitter to prevent thundering herd
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return delay + jitter
}

func (a *API) request(address string, data []byte) []byte {
	encodedData := []byte(base64.StdEncoding.EncodeToString(data))
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", address, bytes.NewBuffer(encodedData))
		if err != nil {
			output.VerbosePrint(fmt.Sprintf("[-] Failed to create HTTP request: %s", err.Error()))
			return nil
		}
		
		resp, err := a.client.Do(req)
		if err != nil {
			if attempt < maxRetries && isRetryableError(err) {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] HTTP request failed (attempt %d/%d): %s. Retrying in %v", 
					attempt+1, maxRetries+1, err.Error(), delay))
				time.Sleep(delay)
				continue
			}
			output.VerbosePrint(fmt.Sprintf("[-] Failed to perform HTTP request after %d attempts: %s", attempt+1, err.Error()))
			return nil
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt < maxRetries {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] Failed to read HTTP response (attempt %d/%d): %s. Retrying in %v", 
					attempt+1, maxRetries+1, err.Error(), delay))
				time.Sleep(delay)
				continue
			}
			output.VerbosePrint(fmt.Sprintf("[-] Failed to read HTTP response after %d attempts: %s", attempt+1, err.Error()))
			return nil
		}
		
		// Check for retryable HTTP status codes
		if isRetryableStatusCode(resp.StatusCode) {
			if attempt < maxRetries {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] HTTP request returned status %d (attempt %d/%d). Retrying in %v", 
					resp.StatusCode, attempt+1, maxRetries+1, delay))
				time.Sleep(delay)
				continue
			}
			output.VerbosePrint(fmt.Sprintf("[-] HTTP request returned status %d after %d attempts", resp.StatusCode, attempt+1))
			return nil
		}
		
		decodedBody, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			if attempt < maxRetries {
				delay := calculateRetryDelay(attempt)
				output.VerbosePrint(fmt.Sprintf("[!] Failed to decode HTTP response (attempt %d/%d): %s. Retrying in %v", 
					attempt+1, maxRetries+1, err.Error(), delay))
				time.Sleep(delay)
				continue
			}
			output.VerbosePrint(fmt.Sprintf("[-] Failed to decode HTTP response after %d attempts: %s", attempt+1, err.Error()))
			return nil
		}
		
		// Success - log if this was a retry
		if attempt > 0 {
			output.VerbosePrint(fmt.Sprintf("[+] HTTP request succeeded on attempt %d", attempt+1))
		}
		return decodedBody
	}
	
	// This should never be reached, but included for safety
	return nil
}