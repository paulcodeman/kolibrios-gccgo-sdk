#!/usr/bin/env python3
"""Reproduce the footer failure in upstream Go, then test the exact UI patch."""
from pathlib import Path
import json
import os
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[2]
UPSTREAM = ROOT / '.build-cache/opencode-upstream'
HOST = ROOT / '.build-cache/opencode/status-host'
STATUS = Path('internal/tui/components/core/status.go')
TEST = r'''package core

import (
    "testing"
    "unicode/utf8"
    "github.com/charmbracelet/lipgloss"
    "github.com/opencode-ai/opencode/internal/config"
    "github.com/opencode-ai/opencode/internal/llm/models"
    "github.com/opencode-ai/opencode/internal/session"
    "github.com/opencode-ai/opencode/internal/tui/util"
    "github.com/spf13/viper"
)

func TestStatusWidth(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    t.Setenv("XDG_CONFIG_HOME", t.TempDir())
    id := models.ModelID("footer.fixture")
    models.SupportedModels[id] = models.Model{ID:id, Name:"fixture", Provider:models.ProviderOpenAI,
        APIModel:"fixture", ContextWindow:200000, DefaultMaxTokens:4096}
    viper.Set("providers.openai.apiKey", "noncredential-fixture")
    for _, agent := range []string{"coder","title","task","summarizer"} {
        viper.Set("agents."+agent+".model", string(id))
    }
    if _, err := config.Load(t.TempDir(), false); err != nil { t.Fatal(err) }
    getHelpWidget() // NewStatusCmp initializes the same widget.
    helpWidget = getHelpWidget()
    for _, name := range []string{"Zen: Big Pickle", "Zen: Nemotron 3.5 Lightning Free", "Zen: Русская модель 中文"} {
        model := models.SupportedModels[id]; model.Name = name; models.SupportedModels[id] = model
        for _, width := range []int{40,60,80,120} {
            for _, notification := range []string{"Model changed to "+name, "Модель изменена: "+name} {
                m := statusCmp{width:width, session:session.Session{ID:"fixture", PromptTokens:28800},
                    info:util.InfoMsg{Msg:notification, Type:util.InfoTypeInfo}}
                view := m.View()
                if lipgloss.Height(view) != 1 || lipgloss.Width(view) > width || !utf8.ValidString(view) {
                    t.Errorf("width=%d model=%q: height=%d cells=%d validUTF8=%v", width,name,
                        lipgloss.Height(view),lipgloss.Width(view),utf8.ValidString(view))
                }
            }
        }
    }
}
'''


def main():
    HOST.mkdir(parents=True, exist_ok=True)
    shutil.copytree(UPSTREAM, HOST, dirs_exist_ok=True,
                    ignore=shutil.ignore_patterns('.git', 'vendor'))
    (HOST / STATUS.parent / 'status_port_test.go').write_text(TEST, encoding='utf-8')
    environment = os.environ.copy()
    environment['GOMAXPROCS'] = '2'
    environment.pop('LOCAL_ENDPOINT', None)
    outcomes = {}
    for label in ('before', 'after'):
        if label == 'after':
            shutil.copyfile(ROOT / '.build-cache/opencode/third_party/github.com/opencode-ai/opencode' / STATUS,
                            HOST / STATUS)
        result = subprocess.run(['go', 'test', '-p', '2', '-count=1', './internal/tui/components/core',
                                 '-run', '^TestStatusWidth$'], cwd=HOST, env=environment,
                                text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        (HOST / (label + '.log')).write_text(result.stdout, encoding='utf-8')
        outcomes[label] = {'exit_code': result.returncode}
        print(label + ': ' + result.stdout[-1800:], flush=True)
    assert outcomes['before']['exit_code'] != 0, 'the upstream regression was not reproduced'
    assert 'height=' in (HOST / 'before.log').read_text(encoding='utf-8'), 'baseline failed before reaching the footer'
    assert outcomes['after']['exit_code'] == 0, (HOST / 'after.log').read_text(encoding='utf-8')
    (HOST / 'validation.json').write_text(json.dumps({'cases':24, 'outcomes':outcomes,
        'upstream_regression_reproduced':True, 'fixed_footer_one_row':True,
        'within_terminal_width':True, 'valid_Unicode':True}, indent=2), encoding='utf-8')
    print('PASS: upstream reproduced; corrected footer stays on one row at 40/60/80/120 columns')


if __name__ == '__main__':
    main()
