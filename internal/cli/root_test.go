package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "md2wiki" {
		t.Errorf("Use = %q, want %q", cmd.Use, "md2wiki")
	}
	if cmd.Short == "" {
		t.Error("Short는 비어 있으면 안 됨")
	}
	if cmd.RunE == nil && cmd.Run == nil && !cmd.HasSubCommands() {
		t.Error("root 커맨드는 RunE/Run 또는 서브커맨드가 있어야 함")
	}
}

func TestNewRootCmd_HasInitCommand(t *testing.T) {
	cmd := NewRootCmd()
	found := false
	for _, c := range cmd.Commands() {
		if c.Name() == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root 커맨드에 init 서브커맨드가 등록되어야 함")
	}
}

func TestNewRootCmd_HelpOnNoArgs(t *testing.T) {
	cmd := NewRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "md2wiki") {
		t.Errorf("help 출력에 %q 가 포함되어야 함, got:\n%s", "md2wiki", buf.String())
	}
}
