package main

import (
"io"  
"fmt"
    "os"
    "testing"
    "net/http"
    "net/http/httptest"
    "strings"
    "sync"
)
// roundTripFunc is a helper type to create a custom RoundTripper for testing.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
    return f(req)
}

// TestBuildDomains tests the BuildDomains function to ensure that it correctly concatenates the username with the expected TLDs.
func TestBuildDomains(t *testing.T) {
    username := "test"
    domains := BuildDomains(username)

    // Expected TLDs from the BuildDomains function.
    expectedTlds := []string{
    ".com",
    ".net",
    ".org",
    ".biz",
    ".info",
    ".name",
    ".pro",
    ".cat",
    ".co",
    ".me",
    ".io",
    ".tech",
    ".dev",
    ".app",
    ".shop",
    ".fail",
    ".xyz",
    ".blog",
    ".portfolio",
    ".store",
    ".online",
    ".about",
    ".space",
    ".lol",
    ".fun",
    ".social",
    }

    if len(domains) != len(expectedTlds) {
    t.Errorf("Expected %d domains, got %d", len(expectedTlds), len(domains))
    }

    for i, tld := range expectedTlds {
    expectedDomain := username + tld
    if domains[i] != expectedDomain {
    t.Errorf("Expected domain %s, got %s", expectedDomain, domains[i])
    }
    }
}
// TestBuildURL tests the BuildURL function to ensure that it correctly replaces the placeholder "{}" with the provided username.
func TestBuildURL(t *testing.T) {
    baseURL := "http://example.com/{}"
    username := "john"
    expected := "http://example.com/john"

    result := BuildURL(baseURL, username)
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}

func TestDeleteOldFile(t *testing.T) {
    username := "tempuser"
    filename := username + ".txt"

    // Create a temporary file to simulate an existing file.
    f, err := os.Create(filename)
    if err != nil {
        t.Fatalf("Failed to create temporary file: %v", err)
    }
    f.WriteString("temporary content")
    f.Close()

    // Call DeleteOldFile to remove the file.
    DeleteOldFile(username)

    // Check that the file does not exist.
    if _, err := os.Stat(filename); !os.IsNotExist(err) {
        t.Errorf("Expected file %s to be deleted, but it still exists", filename)
    }
}
// TestMakeRequestWithProfilePresence tests that when the response body contains the profile indicator,
func TestMakeRequestWithProfilePresence(t *testing.T) {
    // Create a test HTTP server that returns a body containing the indicator phrase "exists"
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Profile exists here."))
    }))
    defer ts.Close()

    // Reset the global count and remove any existing file for the test username
    count.Store(0)
    username := "testuser"
    DeleteOldFile(username)

    // Prepare a Website object with ErrorMsg set to the indicator phrase "exists"
    website := Website{
        Name:            "TestSite",
        BaseURL:         ts.URL, // not used in MakeRequestWithProfilePresence, but required
        ErrorMsg:        "exists",
        FollowRedirects: false,
        UserAgent:       "",
        Cookies:         nil,
    }

    // Invoke the function which should write to file and increment count if the indicator is found
    MakeRequestWithProfilePresence(website, ts.URL, username)

    // Verify that the count has increased to 1
    if count.Load() != 1 {
        t.Errorf("Expected count to be 1, got %d", count.Load())
    }

    // Verify that the file was created and that it contains the expected URL
    content, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    if !strings.Contains(string(content), ts.URL) {
        t.Errorf("Expected file content to contain %s, got %s", ts.URL, string(content))
    }
}
// TestCrackHash tests the CrackHash function by mocking the weakpass API response.
func TestCrackHash(t *testing.T) {
    // Save the original transport and restore it at the end
    originalTransport := http.DefaultTransport
    defer func() { http.DefaultTransport = originalTransport }()

    // Override the default HTTP transport with our custom RoundTripper.
    http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
        expectedURL := "https://weakpass.com/api/v1/search/abc123.json"
        if req.URL.String() == expectedURL {
            resBody := `{"type": "weak", "hash": "abc123", "pass": "password123"}`
            return &http.Response{
                StatusCode: 200,
                Body: io.NopCloser(strings.NewReader(resBody)),
                Header: make(http.Header),
            }, nil
        }
        return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
    })

    // Call CrackHash with a known hash.
    pass := CrackHash("abc123")
    if pass != "password123" {
        t.Errorf("Expected password 'password123', got '%s'", pass)
    }
}
// TestMakeRequestWithErrorCode tests that when the response status code does not match the website's expected error code,
func TestMakeRequestWithErrorCode(t *testing.T) {
    // Reset the global count and delete any old file for the test username.
    count.Store(0)
    username := "errorcodetest"
    DeleteOldFile(username)

    // Create a test HTTP server that always returns status 200.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }))
    defer ts.Close()

    // Prepare a Website object with ErrorCode set to 404 (which does not match the returned 200).
    website := Website{
        Name:            "ErrorCodeTest",
        BaseURL:         ts.URL + "/{}",
        ErrorType:       "status_code",
        ErrorCode:       404,
        FollowRedirects: false,
        UserAgent:       "",
        Cookies:         nil,
    }

    // Build the URL and call MakeRequestWithErrorCode.
    url := BuildURL(website.BaseURL, username)
    MakeRequestWithErrorCode(website, url, username)

    // Check that the global count has increased.
    if count.Load() != 1 {
        t.Errorf("Expected global count to be 1, got %d", count.Load())
    }

    // Check that the file for the given username contains the expected URL.
    expectedURL := BuildURL(website.BaseURL, username)
    content, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    if !strings.Contains(string(content), expectedURL) {
        t.Errorf("Expected file content to contain %s, got %s", expectedURL, string(content))
    }
}
// TestMakeRequestWithErrorMsg tests that when the response body does not contain the error message,
func TestMakeRequestWithErrorMsg(t *testing.T) {
    // Reset global count and remove any file for the test username.
    count.Store(0)
    username := "errormsgtest"
    DeleteOldFile(username)

    // Create an HTTP test server that returns a body WITHOUT the error message indicator.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        // Body does not include the error message "error_not_found".
        w.Write([]byte("This page shows a valid profile"))
    }))
    defer ts.Close()

    // Setup a Website object with ErrorMsg set to the phrase we expect in the failure case.
    website := Website{
        Name:            "TestErrorMsg",
        BaseURL:         ts.URL + "/{}",
        ErrorMsg:        "error_not_found",
        FollowRedirects: false,
        UserAgent:       "",
        Cookies:         nil,
    }

    // Build the URL and call the function; since the response body does not contain the error message,
    // MakeRequestWithErrorMsg should write the URL to file and increase the global count.
    url := BuildURL(website.BaseURL, username)
    MakeRequestWithErrorMsg(website, url, username)

    // Verify that the global count has increased to 1.
    if count.Load() != 1 {
        t.Errorf("Expected global count to be 1, got %d", count.Load())
    }

    // Verify that the file for the username contains the expected URL.
    data, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    expectedURL := BuildURL(website.BaseURL, username)
    if !strings.Contains(string(data), expectedURL) {
        t.Errorf("Expected file content to contain %s, got %s", expectedURL, string(data))
    }
}
// TestMakeRequestWithResponseURL verifies that when the final request URL does not match the expected formatted ResponseURL,
func TestMakeRequestWithResponseURL(t *testing.T) {
    // Create an HTTP test server that always returns a 200 OK.
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        // Simply respond with a plain text body.
        w.Write([]byte("OK"))
    }))
    defer ts.Close()

    // Reset the global counter and delete any previous file for the test username.
    count.Store(0)
    username := "responseTest"
    DeleteOldFile(username)

    // Prepare a Website instance where the BaseURL returns a valid profile,
    // but the ResponseURL (used to compare) will be different.
    website := Website{
        Name:            "TestResponseURL",
        BaseURL:         ts.URL + "/{}",       // This will become ts.URL + "/" + username.
        ResponseURL:     ts.URL + "/different/{}", // This does not match the actual returned URL.
        FollowRedirects: false,
        UserAgent:       "",
        Cookies:         nil,
    }

    // Build the URL from the BaseURL and username.
    url := BuildURL(website.BaseURL, username) // expected to be ts.URL + "/" + username.

    // Call the function under test.
    MakeRequestWithResponseURL(website, url, username)

    // Verify that the global count incremented to 1.
    if count.Load() != 1 {
        t.Errorf("Expected count to be 1, got %d", count.Load())
    }

    // Verify that the file contains the expected URL.
    // The function writes the BaseURL (with username) to the file.
    expectedURL := BuildURL(website.BaseURL, username)
    data, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    if !strings.Contains(string(data), expectedURL) {
        t.Errorf("Expected file content to contain %s, got %s", expectedURL, string(data))
    }
}
// TestWriteToFile tests the WriteToFile function to ensure that it writes data to a file and appends on subsequent calls.
func TestWriteToFile(t *testing.T) {
    username := "writetest"
    // Ensure the temporary file does not exist
    DeleteOldFile(username)
    content := "TestContent\n"
    // Write content twice to verify appending
    WriteToFile(username, content)
    WriteToFile(username, content)
    data, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    expected := content + content
    if string(data) != expected {
        t.Errorf("Expected file content %q, got %q", expected, string(data))
    }
    // Cleanup the file after test
    os.Remove(username + ".txt")
    }
// TestSearchUnknownErrorType tests the Search function when provided with a website having an unknown ErrorType.
func TestSearchUnknownErrorType(t *testing.T) {
    username := "searchtest"
    DeleteOldFile(username)
    count.Store(0)
    data := Data{
        Websites: []Website{
            {
                Name:            "UnknownSite",
                BaseURL:         "http://example.com/{}",
                ErrorType:       "unknown",
                FollowRedirects: false,
                UserAgent:       "",
                Cookies:         nil,
            },
        },
    }
    var wg sync.WaitGroup
    wg.Add(len(data.Websites))
    Search(data, username, false, &wg)
    wg.Wait()
    if count.Load() != 1 {
        t.Errorf("Expected count to be 1, got %d", count.Load())
    }
    expectedURL := BuildURL("http://example.com/{}", username)
    fileData, err := os.ReadFile(username + ".txt")
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    if !strings.Contains(string(fileData), "[?] "+expectedURL) {
        t.Errorf("Expected file content to contain '[?] %s', got %s", expectedURL, string(fileData))
    }
}