package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"time"
)

type codexRuntimeOptions struct {
	Source           string             `json:"source"`
	LoadedAt         string             `json:"loaded_at"`
	Models           []codexModelOption `json:"models"`
	ReasoningEfforts []string           `json:"reasoning_efforts"`
	DefaultModel     string             `json:"default_model,omitempty"`
	DefaultReasoning string             `json:"default_reasoning,omitempty"`
}

type codexModelOption struct {
	Model              string   `json:"model"`
	DisplayName        string   `json:"display_name"`
	DefaultReasoning   string   `json:"default_reasoning,omitempty"`
	SupportedReasoning []string `json:"supported_reasoning,omitempty"`
	IsDefault          bool     `json:"is_default"`
}

var codexOptionsFetcher = fetchCodexRuntimeOptions

func fetchCodexRuntimeOptions(ctx context.Context) (codexRuntimeOptions, error) {
	runCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "codex", "app-server", "--listen", "stdio://")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("open stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("open stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("start codex app-server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stderr)
		_ = cmd.Wait()
	}()

	send := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := stdin.Write(append(b, '\n')); err != nil {
			return err
		}
		return nil
	}

	if err := send(map[string]any{
		"id":     1,
		"method": "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{"name": "knit", "version": "1.0"},
			"capabilities": map[string]any{
				"experimentalApi": true,
			},
		},
	}); err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("send initialize: %w", err)
	}

	sc := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 2*1024*1024)
	waitID := func(target int) (map[string]any, error) {
		for sc.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
				continue
			}
			id, hasID := msg["id"]
			if !hasID {
				continue
			}
			switch v := id.(type) {
			case float64:
				if int(v) == target {
					return msg, nil
				}
			case int:
				if v == target {
					return msg, nil
				}
			}
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no response for id=%d", target)
	}

	if _, err := waitID(1); err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("initialize response: %w", err)
	}
	if err := send(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("send initialized: %w", err)
	}
	if err := send(map[string]any{
		"id":     2,
		"method": "model/list",
		"params": map[string]any{"includeHidden": false, "limit": 200},
	}); err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("send model/list: %w", err)
	}

	resp, err := waitID(2)
	if err != nil {
		return codexRuntimeOptions{}, fmt.Errorf("model/list response: %w", err)
	}
	resAny, ok := resp["result"]
	if !ok {
		return codexRuntimeOptions{}, fmt.Errorf("model/list missing result")
	}
	resMap, ok := resAny.(map[string]any)
	if !ok {
		return codexRuntimeOptions{}, fmt.Errorf("model/list malformed result")
	}
	dataAny, _ := resMap["data"].([]any)
	if len(dataAny) == 0 {
		return codexRuntimeOptions{}, fmt.Errorf("no models returned from codex")
	}

	out := codexRuntimeOptions{
		Source:   "codex_cli",
		LoadedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	reasonSet := map[string]struct{}{}
	for _, item := range dataAny {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		model, _ := m["model"].(string)
		if model == "" {
			continue
		}
		displayName, _ := m["displayName"].(string)
		if displayName == "" {
			displayName = model
		}
		isDefault, _ := m["isDefault"].(bool)
		defaultReasoning, _ := m["defaultReasoningEffort"].(string)
		supported := []string{}
		if sAny, ok := m["supportedReasoningEfforts"].([]any); ok {
			for _, s := range sAny {
				sMap, ok := s.(map[string]any)
				if !ok {
					continue
				}
				r, _ := sMap["reasoningEffort"].(string)
				if r == "" {
					continue
				}
				reasonSet[r] = struct{}{}
				supported = append(supported, r)
			}
		}
		out.Models = append(out.Models, codexModelOption{
			Model:              model,
			DisplayName:        displayName,
			DefaultReasoning:   defaultReasoning,
			SupportedReasoning: supported,
			IsDefault:          isDefault,
		})
		if isDefault {
			out.DefaultModel = model
			if out.DefaultReasoning == "" {
				out.DefaultReasoning = defaultReasoning
			}
		}
	}
	for k := range reasonSet {
		out.ReasoningEfforts = append(out.ReasoningEfforts, k)
	}
	sort.Strings(out.ReasoningEfforts)
	sort.SliceStable(out.Models, func(i, j int) bool {
		if out.Models[i].IsDefault != out.Models[j].IsDefault {
			return out.Models[i].IsDefault
		}
		return out.Models[i].DisplayName < out.Models[j].DisplayName
	})
	if len(out.ReasoningEfforts) == 0 {
		out.ReasoningEfforts = []string{"low", "medium", "high"}
	}
	return out, nil
}
