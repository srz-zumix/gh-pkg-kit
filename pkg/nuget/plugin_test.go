package nuget

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type pluginMessage struct {
	RequestID string          `json:"RequestId"`
	Type      string          `json:"Type"`
	Method    string          `json:"Method"`
	Payload   json.RawMessage `json:"Payload,omitempty"`
}

func TestCredentialProviderDotnetRestore(t *testing.T) {
	if os.Getenv("GH_PKG_KIT_TEST_DOTNET") != "1" || runtime.GOOS == "windows" {
		t.Skip("set GH_PKG_KIT_TEST_DOTNET=1 with .NET SDK 10+ to run the local restore integration test")
	}
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		t.Fatal(err)
	}
	sdkVersion, err := exec.Command(dotnet, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	major, _, _ := strings.Cut(strings.TrimSpace(string(sdkVersion)), ".")
	if n, err := strconv.Atoi(major); err != nil || n < 10 {
		t.Skipf("the local restore integration test requires .NET SDK 10+ (found %s)", strings.TrimSpace(string(sdkVersion)))
	}
	dir := t.TempDir()
	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	gh := filepath.Join(dir, "gh")
	write(gh, "#!/bin/sh\nprintf '%s\\n' local-test-token\n", 0755)
	_, err = InstallCredentialProvider(context.Background(), filepath.Join(dir, ".nuget", "plugins", "netcore"), false)
	if err != nil {
		t.Fatal(err)
	}
	var packageData bytes.Buffer
	archive := zip.NewWriter(&packageData)
	for name, content := range map[string]string{
		"CredentialProviderTest.nuspec":           `<package><metadata><id>CredentialProviderTest</id><version>1.0.0</version><authors>test</authors><description>Local authentication test</description><packageTypes><packageType name="DotnetTool" /></packageTypes></metadata></package>`,
		"tools/net8.0/any/DotnetToolSettings.xml": `<DotNetCliTool Version="1"><Commands><Command Name="credential-provider-test" EntryPoint="Test.dll" Runner="dotnet" /></Commands></DotNetCliTool>`,
		"tools/net8.0/any/Test.dll":               "test entrypoint (not executed)",
	} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	var authenticated atomic.Bool
	var feedURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != credentialUsername || password != "local-test-token" {
			response.Header().Set("WWW-Authenticate", `Basic realm="local-test"`)
			response.WriteHeader(http.StatusUnauthorized)
			return
		}
		authenticated.Store(true)
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/_registry/nuget/index.json":
			fmt.Fprintf(response, `{"version":"3.0.0","resources":[{"@id":%q,"@type":"PackageBaseAddress/3.0.0"},{"@id":%q,"@type":"RegistrationsBaseUrl/3.6.0"}]}`, feedURL+"/_registry/nuget/flat/", feedURL+"/_registry/nuget/registration/")
		case "/_registry/nuget/registration/credentialprovidertest/index.json":
			fmt.Fprintf(response, `{"count":1,"items":[{"count":1,"lower":"1.0.0","upper":"1.0.0","items":[{"catalogEntry":{"id":"CredentialProviderTest","version":"1.0.0","authors":"test","description":"Local authentication test","listed":true},"packageContent":%q}]}]}`, feedURL+"/_registry/nuget/flat/credentialprovidertest/1.0.0/credentialprovidertest.1.0.0.nupkg")
		case "/_registry/nuget/flat/credentialprovidertest/index.json":
			fmt.Fprint(response, `{"versions":["1.0.0"]}`)
		case "/_registry/nuget/flat/credentialprovidertest/1.0.0/credentialprovidertest.1.0.0.nupkg":
			response.Header().Set("Content-Type", "application/octet-stream")
			_, _ = response.Write(packageData.Bytes())
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	feedURL = server.URL
	parsed, _ := url.Parse(feedURL)
	if err := os.Mkdir(filepath.Join(dir, ".config"), 0755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(dir, ".config", "dotnet-tools.json"), `{"version":1,"isRoot":true,"tools":{"credentialprovidertest":{"version":"1.0.0","commands":["credential-provider-test"]}}}`, 0600)
	configPath := filepath.Join(dir, "NuGet.Config")
	write(configPath, fmt.Sprintf(`<configuration><packageSources><clear/><add key="test" value="%s/_registry/nuget/index.json" disableTLSCertificateValidation="true" /></packageSources></configuration>`, feedURL), 0600)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, dotnet, "tool", "restore", "--configfile", configPath, "--no-cache", "--verbosity", "minimal")
	command.Dir = dir
	command.Env = append(os.Environ(), "GH_HOST="+parsed.Hostname(), "GH_ENTERPRISE_TOKEN=local-test-token", "GH_CONFIG_DIR="+dir, "GH_PATH="+gh,
		"HOME="+dir, "DOTNET_CLI_HOME="+dir, "DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1", "DOTNET_CLI_TELEMETRY_OPTOUT=1",
		"NUGET_PACKAGES="+filepath.Join(dir, "packages"), "DOTNET_GENERATE_ASPNET_CERTIFICATE=false", "DOTNET_NOLOGO=true",
		"NUGET_PLUGIN_PATHS=", "NUGET_NETCORE_PLUGIN_PATHS=", "NUGET_NETFX_PLUGIN_PATHS=", "NUGET_HTTP_CACHE_PATH="+filepath.Join(dir, "http-cache"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("dotnet restore: %v\n%s", err, output)
	}
	if !authenticated.Load() {
		t.Fatal("NuGet did not authenticate with the credential provider")
	}
	if _, err := os.Stat(filepath.Join(dir, "packages", "credentialprovidertest", "1.0.0", "credentialprovidertest.1.0.0.nupkg")); err != nil {
		t.Fatalf("package not restored: %v", err)
	}
}

func TestCredentialProviderProtocol(t *testing.T) {
	process, requests, responses := startTestProvider(t)
	decoder := json.NewDecoder(responses)
	encoder := json.NewEncoder(requests)
	var handshake pluginMessage
	if err := decoder.Decode(&handshake); err != nil {
		t.Fatal(err)
	}
	send := func(id, kind, method string, payload any) {
		t.Helper()
		data, _ := json.Marshal(payload)
		if err := encoder.Encode(pluginMessage{id, kind, method, data}); err != nil {
			t.Fatal(err)
		}
	}
	receive := func(id string) map[string]any {
		t.Helper()
		var response pluginMessage
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.RequestID != id || response.Type != "Response" {
			t.Fatalf("unexpected response: %+v", response)
		}
		var payload map[string]any
		if err := json.Unmarshal(response.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	send(handshake.RequestID, "Response", "Handshake", map[string]string{"ResponseCode": "Success", "ProtocolVersion": "2.0.0"})
	send("client-handshake", "Request", "Handshake", map[string]string{"ProtocolVersion": "2.0.0", "MinimumProtocolVersion": "1.0.0"})
	if payload := receive("client-handshake"); payload["ProtocolVersion"] != "2.0.0" {
		t.Fatal(payload)
	}
	send("init", "Request", "Initialize", map[string]any{"ClientVersion": "6.14.3", "Culture": "en-US", "RequestTimeout": "00:00:05"})
	receive("init")
	send("claims", "Request", "GetOperationClaims", map[string]any{})
	if payload := receive("claims"); len(payload["Claims"].([]any)) != 1 {
		t.Fatal(payload)
	}
	send("source-claims", "Request", "GetOperationClaims", map[string]any{"PackageSourceRepository": "https://nuget.pkg.github.com/owner/index.json"})
	if payload := receive("source-claims"); len(payload["Claims"].([]any)) != 0 {
		t.Fatal(payload)
	}
	send("auth", "Request", "GetAuthenticationCredentials", map[string]any{"Uri": "https://nuget.pkg.github.com/owner/index.json", "IsRetry": true, "IsNonInteractive": true, "CanShowDialog": false})
	if payload := receive("auth"); payload["Password"] != "fake-token" {
		t.Fatal("missing fake token")
	}
	for index, test := range []struct{ uri, code string }{
		{"https://nuget.ghe.example/owner/index.json", "Success"},
		{"https://ghe.example/_registry/nuget/owner/index.json", "Success"},
		{"https://api.nuget.org/v3/index.json", "NotFound"},
		{"http://nuget.pkg.github.com/owner/index.json", "NotFound"},
		{"https://nuget.pkg.github.com.evil.example/owner/index.json", "NotFound"},
		{"https://user@nuget.pkg.github.com/owner/index.json", "NotFound"},
		{"https://nuget.missing.example/owner/index.json", "NotFound"},
	} {
		id := fmt.Sprintf("uri-%d", index)
		send(id, "Request", "GetAuthenticationCredentials", map[string]any{"Uri": test.uri, "IsRetry": false, "IsNonInteractive": true, "CanShowDialog": false})
		payload := receive(id)
		if payload["ResponseCode"] != test.code || (test.code == "NotFound" && payload["Password"] != nil) {
			t.Fatalf("incorrect response code for %s", test.uri)
		}
	}
	send("slow", "Request", "GetAuthenticationCredentials", map[string]any{"Uri": "https://nuget.slow.example/owner/index.json", "IsRetry": false, "IsNonInteractive": true, "CanShowDialog": false})
	var progress pluginMessage
	if err := decoder.Decode(&progress); err != nil || progress.Type != "Progress" || progress.RequestID != "slow" {
		t.Fatalf("expected progress for token lookup: %v", err)
	}
	send("slow", "Cancel", "GetAuthenticationCredentials", nil)
	var cancellation pluginMessage
	if err := decoder.Decode(&cancellation); err != nil || cancellation.Type != "Cancel" || cancellation.RequestID != "slow" {
		t.Fatalf("invalid cancellation response: %+v, %v", cancellation, err)
	}
	send("close", "Request", "Close", nil)
	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialProviderRejectsInvalidHandshake(t *testing.T) {
	process, requests, responses := startTestProvider(t)
	decoder := json.NewDecoder(responses)
	var handshake pluginMessage
	if err := decoder.Decode(&handshake); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(requests, `{"RequestId":"old","Type":"Request","Method":"Handshake","Payload":{"ProtocolVersion":"1.0.0","MinimumProtocolVersion":"1.0.0"}}`+"\n"); err != nil {
		t.Fatal(err)
	}
	var response pluginMessage
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(response.Payload), "Success") {
		t.Fatal("must not negotiate protocol v1")
	}
	encoder := json.NewEncoder(requests)
	payload, _ := json.Marshal(map[string]string{"ResponseCode": "Error"})
	if err := encoder.Encode(pluginMessage{RequestID: handshake.RequestID, Type: "Response", Method: "Handshake", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	requests.Close()
	if err := process.Wait(); err == nil {
		t.Fatal("incompatible handshake must terminate the provider")
	}
}

func startTestProvider(t *testing.T) (*exec.Cmd, io.WriteCloser, io.ReadCloser) {
	t.Helper()
	if os.Getenv("GH_PKG_KIT_TEST_DOTNET") != "1" || runtime.GOOS == "windows" {
		t.Skip("set GH_PKG_KIT_TEST_DOTNET=1 with .NET SDK 8+ to run .NET protocol tests")
	}
	dir := t.TempDir()
	gh := filepath.Join(dir, "gh")
	script := "#!/bin/sh\ncase \"$2\" in\nstatus) printf '%s\\n' '{\"hosts\":{\"ghe.example\":[],\"missing.example\":[],\"slow.example\":[]}}';;\ntoken) case \"$4\" in\nslow.example) exec node -e 'setInterval(() => {}, 1000)';;\nmissing.example) exit 1;;\n*) printf '%s\\n' fake-token;;\nesac;;\n*) exit 1;;\nesac\n"
	if err := os.WriteFile(gh, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	path, err := InstallCredentialProvider(context.Background(), filepath.Join(dir, "plugins"), false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	process := exec.CommandContext(ctx, "dotnet", path, "-Plugin")
	process.Env = append(os.Environ(), "GH_PATH="+gh, "GH_HOST=slow.example", "NUGET_PLUGIN_ENABLE_LOG=false")
	requests, err := process.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	responses, err := process.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	process.Stderr = io.Discard
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		requests.Close()
		responses.Close()
		if process.ProcessState == nil {
			_ = process.Process.Kill()
			_ = process.Wait()
		}
	})
	return process, requests, responses
}
