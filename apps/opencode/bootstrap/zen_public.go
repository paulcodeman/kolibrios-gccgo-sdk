package main

// Adapt the current OpenCode public-provider catalog to the original Go CLI's
// exported model registry and existing OpenAI-compatible client. Upstream:
// https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/provider/provider.ts
// https://models.dev/api.json
// No upstream application, SDK HTTP implementation or TLS policy is replaced.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/spf13/viper"
)

const zenPublicEndpoint = "https://opencode.ai/zen/v1"

type zenCatalogModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Attachment bool   `json:"attachment"`
	ToolCall   bool   `json:"tool_call"`
	Cost       struct {
		Input  *float64 `json:"input"`
		Output *float64 `json:"output"`
	} `json:"cost"`
	Limit struct {
		Context int64 `json:"context"`
		Output  int64 `json:"output"`
	} `json:"limit"`
	Provider struct {
		NPM string `json:"npm"`
	} `json:"provider"`
}

type zenCatalog struct {
	NPM    string                     `json:"npm"`
	API    string                     `json:"api"`
	Models map[string]zenCatalogModel `json:"models"`
}

// Identify this actual Go OpenCode port truthfully. Public-tier refusals from
// the service are left intact; this adapter does not impersonate a newer CLI.
type zenPublicTransport struct{ base http.RoundTripper }

func (t zenPublicTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Hostname() == "opencode.ai" || request.URL.Hostname() == "models.dev" {
		request = request.Clone(request.Context())
		request.Header.Set("User-Agent", "opencode/go-73ee493-kolibrios")
	}
	return t.base.RoundTrip(request)
}

func zenReadJSON(client *http.Client, address string, target any) error {
	response, err := client.Get(address)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", address, response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(target)
}

func loadZenPublicModels() error {
	if os.Getenv("OPENCODE_ZEN_PUBLIC") != "1" {
		return nil
	}
	http.DefaultTransport = zenPublicTransport{base: http.DefaultTransport}
	client := &http.Client{Timeout: 45 * time.Second}
	var live struct {
		Data []struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := zenReadJSON(client, zenPublicEndpoint+"/models", &live); err != nil {
		return err
	}
	var catalog struct{ OpenCode zenCatalog `json:"opencode"` }
	if err := zenReadJSON(client, "https://models.dev/api.json", &catalog); err != nil {
		return err
	}
	if catalog.OpenCode.API != zenPublicEndpoint || catalog.OpenCode.NPM != "@ai-sdk/openai-compatible" {
		return fmt.Errorf("unsupported Zen catalog protocol")
	}
	available := make(map[string]bool)
	for _, entry := range live.Data {
		available[entry.ID] = true
	}
	selected := make([]models.Model, 0)
	for id, entry := range catalog.OpenCode.Models {
		// Require explicit zero input AND output prices, live availability and
		// tool support. Responses/Anthropic/structured-only models cannot use
		// the original Go CLI's Chat Completions client.
		if !available[id] || entry.Status == "deprecated" || !entry.ToolCall ||
			entry.Cost.Input == nil || entry.Cost.Output == nil ||
			*entry.Cost.Input != 0 || *entry.Cost.Output != 0 ||
			(entry.Provider.NPM != "" && entry.Provider.NPM != "@ai-sdk/openai-compatible") ||
			entry.Limit.Context <= 0 || entry.Limit.Output <= 0 {
			continue
		}
		selected = append(selected, models.Model{
			ID: models.ModelID("local.zen-" + id), Name: "Zen: " + entry.Name,
			Provider: models.ProviderLocal, APIModel: id,
			ContextWindow: entry.Limit.Context,
			DefaultMaxTokens: min(int64(4096), entry.Limit.Output, entry.Limit.Context/2),
			SupportsAttachments: entry.Attachment,
			// The old OpenAI client couples CanReason to reasoning_effort and
			// max_completion_tokens. Zen's compatible route accepts max_tokens;
			// reasoning is handled by the remote model's own defaults.
		})
	}
	if len(selected) == 0 {
		return fmt.Errorf("no free compatible models in the current Zen catalog")
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Name < selected[j].Name })
	preferred := selected[0].ID
	for id, model := range models.SupportedModels {
		if model.Provider == models.ProviderLocal {
			delete(models.SupportedModels, id)
		}
	}
	for _, model := range selected {
		models.SupportedModels[model.ID] = model
		if model.APIModel == "space-bunny-free" {
			preferred = model.ID
		}
	}
	// Reuse upstream's configurable compatible provider, including its model
	// picker, streaming, tools, cancellation and SQLite persistence.
	if err := os.Setenv("LOCAL_ENDPOINT", zenPublicEndpoint); err != nil {
		return err
	}
	viper.Set("providers.local.apiKey", "public")
	viper.Set("providers.local.disabled", false)
	for _, agent := range []string{"coder", "summarizer", "task", "title"} {
		viper.SetDefault("agents."+agent+".model", preferred)
	}
	models.ProviderPopularity[models.ProviderLocal] = 1
	// Keep a readable record of exactly the models and context limits used.
	if cache, err := os.UserCacheDir(); err == nil {
		directory := filepath.Join(cache, "opencode")
		if os.MkdirAll(directory, 0o755) == nil {
			if data, err := json.MarshalIndent(selected, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(directory, "zen-public-models.json"), data, 0o644)
			}
		}
	}
	return nil
}
